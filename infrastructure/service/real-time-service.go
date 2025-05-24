package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		LabelSelector: getLabelSelector(ctx, cfg),
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
						err := streamLogs(
							c,
							clientset,
							getDir(ctx, cfg),
							cfg.Namespace,
							pName,
						)
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

// streamLogs streams the logs from a specified K8s pod in real-time, writing them to a local file
// and processing each log line asynchronously. It listens for context cancellation to gracefully stop streaming.
// The function takes a context for cancellation, a Kubernetes clientset, the directory to store logs, the namespace,
// and the pod name. It returns an error if any occurs during log streaming, file operations, or log processing.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - clientset: Kubernetes clientset to interact with the cluster.
//   - dir: Directory path where the log file will be stored.
//   - namespace: Namespace of the target pod.
//   - podName: Name of the pod to stream logs from.
//
// Returns:
//   - error: An error if streaming, file writing, or log processing fails; otherwise, nil.
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

	reader := bufio.NewReader(stream)
	filename := filepath.Join(dir, podName+".log")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error creando archivo log pod %s: %w", podName, err)
	}
	defer file.Close()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Cancelando streamLogs para pod %s", podName)
			return nil
		default:
			lineBytes, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return fmt.Errorf("stream cerrado para pod %s", podName)
				}
				return fmt.Errorf("error leyendo log pod %s: %w", podName, err)
			}
			line := strings.TrimSuffix(lineBytes, "\n")

			if _, wErr := file.WriteString(line + "\n"); wErr != nil {
				return fmt.Errorf("error escribiendo log pod %s: %w", podName, wErr)
			}

			go repository.SaveLog(ctx, line)
		}
	}
}

// Take a path for the target directory, taking into account that
// the first option it's witch come from flag.
func getDir(ctx context.Context, cfg *domain.Config) string {
	if ctx.Value(domain.CtxKeyType("dir")) != "" {
		return ctx.Value(domain.CtxKeyType("dir")).(string)
	} else {
		return cfg.LogDirectory
	}
}

func getLabelSelector(ctx context.Context, cfg *domain.Config) string {
	srvName := ctx.Value(domain.CtxKeyType("srvName"))
	if srvName != "" {
		return srvName.(string)
	} else if srvName == "*" {
		return ""
	} else {
		return cfg.LabelSelector
	}
}
