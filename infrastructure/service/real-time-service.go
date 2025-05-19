package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/repository"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

func RealTimeProcess(ctx context.Context, cfg *domain.Config) {
	clientset, err := GetKubernetesClient()
	if err != nil {
		log.Fatalf("Error creando cliente: %v", err)
	}

	// Mapa para controlar descargas activas de logs: podName -> cancelFunc
	activeLogs := make(map[string]context.CancelFunc)
	var mu sync.Mutex

	watcher, err := clientset.CoreV1().Pods(cfg.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: cfg.LabelSelector,
	})
	if err != nil {
		log.Fatalf("Error creando watcher: %v", err)
	}
	defer watcher.Stop()

	log.Println("Observando pods...")

	for {
		select {
		case <-ctx.Done():
			// Cancelar todos los logs activos
			mu.Lock()
			for pod, cancelFunc := range activeLogs {
				log.Printf("Cancelando log stream de pod %s", pod)
				cancelFunc()
			}
			mu.Unlock()
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				log.Println("Watcher cerrado, terminando.")
				return
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				log.Println("Evento no es un Pod")
				continue
			}

			podName := pod.Name

			mu.Lock()
			cancelFunc, isActive := activeLogs[podName]
			mu.Unlock()

			switch event.Type {
			case watch.Added, watch.Modified:
				// Si el pod está Running y no estamos descargando logs para él, iniciar
				if pod.Status.Phase == corev1.PodRunning && !isActive {
					log.Printf("Pod %s está Running, iniciando descarga de logs", podName)
					// Crear contexto para cancelar lectura de logs
					logCtx, logCancel := context.WithCancel(ctx)

					mu.Lock()
					activeLogs[podName] = logCancel
					mu.Unlock()

					go func(pName string, c context.Context) {
						err := streamLogs(c, clientset, cfg.LogDirectory, cfg.Namespace, pName)
						if err != nil {
							log.Printf("Error en streamLogs pod %s: %v", pName, err)
						}
						// Cuando termina la descarga, limpiar del mapa
						mu.Lock()
						delete(activeLogs, pName)
						mu.Unlock()
					}(podName, logCtx)
				}
			case watch.Deleted:
				// Cuando un pod se elimina, cancelar la descarga de logs si estaba activa
				if isActive {
					log.Printf("Pod %s eliminado, cancelando descarga de logs", podName)
					cancelFunc()
					mu.Lock()
					delete(activeLogs, podName)
					mu.Unlock()
				}
			}
		}
	}
}

func streamLogs(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	dir, namespace, podName string,
) error {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Follow: true,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("error abriendo stream logs pod %s: %w", podName, err)
	}
	defer stream.Close()

	filename := filepath.Join(dir, podName+".log")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error creando archivo log pod %s: %w", podName, err)
	}
	defer file.Close()

	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Cancelando streamLogs para pod %s", podName)
			return nil
		default:
			n, err := stream.Read(buf)
			if n > 0 {
				_, wErr := file.Write(buf[:n])
				if wErr != nil {
					return fmt.Errorf("error escribiendo log pod %s: %w", podName, wErr)
				}

				go func() {
					line := string(buf[:n])
					repository.SaveLog(ctx, line)
				}()
			}
			if err != nil {
				if err == io.EOF {
					log.Printf("Stream cerrado para pod %s", podName)
					return nil
				}
				return fmt.Errorf("error leyendo stream pod %s: %w", podName, err)
			}
		}
	}
}
