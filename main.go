package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Config struct {
	Namespace     string `json:"namespace"`
	LabelSelector string `json:"labelSelector"`
	LogDirectory  string `json:"logDirectory"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
}

var timeRegexes = []*regexp.Regexp{
	regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2}))\]`),                 // Ej: [2025-05-15T17:22:59-0500]
	regexp.MustCompile(`"timestamp"\s*:\s*"(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2}))"`), // Ej: "timestamp":"2025-05-15T17:22:59.820-05:00"
}
var activeLogStreams = make(map[string]context.CancelFunc)

func loadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	// Flags
	realtime := flag.Bool("realtime", false, "Habilita la lectura en tiempo real de logs")
	btimes := flag.Bool("btimes", false, "Habilita la descarga de logs entre dos tiempos definidos")
	startFlag := flag.String("start", "", "Hora de inicio en formato HH:MM (opcional, también puede ir en config)")
	endFlag := flag.String("end", "", "Hora de fin en formato HH:MM (opcional, también puede ir en config)")
	flag.Parse()

	if *realtime && *btimes {
		log.Fatal("No puedes usar -realtime y -btimes al mismo tiempo.")
	}
	if !*realtime && !*btimes {
		log.Fatal("Debes usar al menos uno de los flags: -realtime o -btimes.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Manejo de señales para cerrar la app con CTRL+C
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		cancel()
	}()

	// Leer archivo de configuración
	cfg, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	errLogDir := ensureDir(cfg.LogDirectory)
	if errLogDir != nil {
		log.Fatalln("Error creating log dir: %v", errLogDir)
	}

	if *realtime {
		log.Println("Download logs in real time.")
		realTimeProcess(ctx, cfg)
	}
	if *btimes {
		log.Println("Download logs between times.")
		var startTimeStr, endTimeStr string

		if *startFlag != "" {
			startTimeStr = *startFlag
			endTimeStr = *endFlag
		} else if cfg.StartTime != "" {
			startTimeStr = cfg.StartTime
			endTimeStr = cfg.EndTime
		} else {
			log.Fatal("Debes proporcionar -start y -end o definirlos en config.json")
		}

		startTime, err := parseHour(startTimeStr)
		if err != nil {
			log.Fatalf("startTime inválido: %v", err)
		}

		var endTime time.Time
		if endTimeStr != "" {
			endTime, err = parseHour(endTimeStr)
			if err != nil {
				log.Fatalf("endTime inválido: %v", err)
			}
		} else {
			endTime = time.Now()
		}

		if endTime.Before(startTime) {
			log.Fatal("endTime no puede ser anterior a startTime")
		}

		log.Printf("Descargando logs entre %s y %s...\n", startTime.Format("15:04"), endTime.Format("15:04"))
		betweenTimesProcess(ctx, cfg, startTime, endTime)
	}
}

func betweenTimesProcess(ctx context.Context, cfg *Config, startTime, endTime time.Time) {
	clientset, err := getKubernetesClient()
	if err != nil {
		log.Fatalf("Error al obtener el cliente de Kubernetes: %v", err)
	}

	pods, err := getPodsByLabel(clientset, cfg)
	if err != nil {
		log.Fatalf("Error al obtener pods: %v", err)
	}

	logDir := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05"))
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatalf("No se pudo crear directorio para logs: %v", err)
	}

	for _, pod := range pods {
		fmt.Printf("Procesando logs para pod %s...\n", pod.Name)

		req := clientset.CoreV1().Pods(cfg.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			SinceTime: &metav1.Time{Time: startTime},
			Follow:    false,
		})

		stream, err := req.Stream(ctx)
		if err != nil {
			log.Printf("Error al obtener logs del pod %s: %v", pod.Name, err)
			continue
		}
		defer stream.Close()

		logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", pod.Name))
		f, err := os.Create(logFile)
		if err != nil {
			log.Printf("No se pudo crear archivo de logs para %s: %v", pod.Name, err)
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()
			logTime, err := extractTimestamp(line)
			if err != nil {
				log.Printf("No se pudo parsear la línea: %s", line)
				continue
			}
			if logTime.After(endTime) {
				break
			}
			f.WriteString(line + "\n")
		}
	}
}

func realTimeProcess(ctx context.Context, cfg *Config) {
	clientset, err := getKubernetesClient()
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

func getKubernetesClient() (*kubernetes.Clientset, error) {
	// Intenta config InCluster
	config, err := rest.InClusterConfig()
	if err != nil {
		// Si falla, intenta con kubeconfig local
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("error creando config Kubernetes: %w", err)
		}
	}
	return kubernetes.NewForConfig(config)
}

func getPodsByLabel(clientset *kubernetes.Clientset, cfg *Config) ([]corev1.Pod, error) {
	podList, err := clientset.CoreV1().Pods(cfg.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: cfg.LabelSelector,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func streamLogs(ctx context.Context, clientset *kubernetes.Clientset, dir, namespace, podName string) error {
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

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// No existe, crear carpeta
		return os.MkdirAll(path, 0755)
	}
	if err != nil {
		// Otro error al intentar acceder
		return err
	}
	if !info.IsDir() {
		// Existe pero no es directorio, error
		return fmt.Errorf("%s ya existe pero no es un directorio", path)
	}
	// Existe y es directorio, todo ok
	return nil
}

func parseHour(h string) (time.Time, error) {
	now := time.Now()
	parsed, err := time.Parse("15:04", h)
	if err != nil {
		return time.Time{}, err
	}
	// Combina la fecha de hoy con la hora proporcionada
	return time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location()), nil
}

func parseLogTimestamp(logLine string) (time.Time, error) {
	if len(logLine) < 20 {
		return time.Time{}, fmt.Errorf("línea muy corta para contener timestamp: %s", logLine)
	}
	// Ejemplo: 2025-05-16T15:04:05.000000000Z
	return time.Parse(time.RFC3339Nano, logLine[:len("2006-01-02T15:04:05.999999999Z")])
}

func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,                    // "2025-05-15T17:22:59-05:00"
		"2006-01-02T15:04:05Z0700",      // "2025-05-15T17:22:59-0500"
		"2006-01-02T15:04:05.000Z0700",  // "2025-05-15T17:22:59.820-0500"
		"2006-01-02T15:04:05.000Z07:00", // "2025-05-15T17:22:59.820-05:00"
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid timestamp format: %s", s)
}

func extractTimestamp(logLine string) (time.Time, error) {
	for _, r := range timeRegexes {
		if match := r.FindStringSubmatch(logLine); match != nil {
			return parseTimestamp(match[1])
		} else {
			log.Printf("No se pudo parsear la línea: %s", logLine)
		}
	}
	return time.Time{}, fmt.Errorf("no timestamp found")
}
