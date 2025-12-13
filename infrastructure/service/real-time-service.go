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
	"time"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/repository"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

func RealTimeProcess(
	ctx context.Context, cfg *domain.Config,
) {
	clientset, err := GetKubernetesClient()
	if err != nil {
		log.Fatalf("Error creando cliente: %v", err)
	}

	// Mapa para controlar descargas activas de logs: podName -> cancelFunc
	activeLogs := make(map[string]context.CancelFunc)
	var mu sync.Mutex

	for { // Bucle externo para reintentar la creación del watcher
		watcher, err := clientset.
			CoreV1().
			Pods(cfg.Namespace).
			Watch(ctx, metav1.ListOptions{
				LabelSelector: getLabelSelector(ctx, cfg),
			})
		if err != nil {
			log.Printf("Error creando watcher, reintentando en 5 segundos: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("Observando pods...")

	eventLoop:
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
				watcher.Stop()
				return
			case event, ok := <-watcher.ResultChan():
				if !ok {
					log.Println("Watcher cerrado, intentando reconectar...")
					watcher.Stop()

					// Cancelar todos los logs activos antes de reconectar
					mu.Lock()
					for pod, cancelFunc := range activeLogs {
						log.Printf("Cancelando log stream de pod %s antes de reconectar", pod)
						cancelFunc()
						delete(activeLogs, pod) // Eliminar del mapa inmediatamente
					}
					mu.Unlock()
					time.Sleep(1 * time.Second) // Dar un momento a las goroutines para limpiar
					break eventLoop             // Salir del bucle de eventos para recrear el watcher
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
								cfg,
							)
							if err != nil && !strings.Contains(err.Error(), "context canceled") {
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
	cfg *domain.Config,
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

	var file *os.File
	var currentSize int64
	maxLogFileSize := int64(cfg.Fsize) * 1024 * 1024

	// Ensure the last file is closed when the function returns
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	for {
		// Check if we need to rotate the file
		if file == nil || currentSize > maxLogFileSize {
			if file != nil {
				file.Close()
			}

			timestamp := time.Now().Format("20060102150405")
			filename := filepath.Join(dir, fmt.Sprintf("%s_%s.log", podName, timestamp))

			var err error
			file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("error creando archivo log pod %s: %w", podName, err)
			}
			currentSize = 0
		}

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

			n, wErr := file.WriteString(line + "\n")
			if wErr != nil {
				return fmt.Errorf("error escribiendo log pod %s: %w", podName, wErr)
			}
			currentSize += int64(n)

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
	var result string
	if srvName == "all" {
		result = ""
	} else if srvName != "" {
		result = fmt.Sprintf("app=%s", srvName.(string))
	} else {
		result = cfg.LabelSelector
	}

	return result
}
