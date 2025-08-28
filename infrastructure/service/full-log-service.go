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
	"k8s.io/client-go/kubernetes"
)

func FullLogProcess(ctx context.Context, cfg *domain.Config) {
	// Crea un cliente SIN timeout para la descarga completa
	clientset, err := GetKubernetesClientWithTimeout(0)
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	pods, err := clientset.CoreV1().Pods(cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: getLabelSelector(ctx, cfg),
	})
	if err != nil {
		log.Fatalf("Error listing pods: %v", err)
	}

	var wg sync.WaitGroup
	for _, pod := range pods.Items {
		wg.Add(1)
		go func(podName string) {
			defer wg.Done()
			log.Printf("Downloading full log for pod %s", podName)
			// Pasa el cliente sin timeout a la función de descarga
			err := downloadFullLog(ctx, clientset, getDir(ctx, cfg), cfg.Namespace, podName)
			if err != nil {
				log.Printf("Error downloading full log for pod %s: %v", podName, err)
			}
		}(pod.Name)
	}
	wg.Wait()
	log.Println("All full logs downloaded.")
}

func downloadFullLog(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	dir, namespace, podName string,
) error {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})

	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("error opening stream for pod %s: %w", podName, err)
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	filename := filepath.Join(dir, podName+"-full.log")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error creating log file for pod %s: %w", podName, err)
	}
	defer file.Close()

	for {
		lineBytes, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading log from pod %s: %w", podName, err)
		}
		line := strings.TrimSuffix(lineBytes, "\n")

		if _, wErr := file.WriteString(line + "\n"); wErr != nil {
			return fmt.Errorf("error writing to log file for pod %s: %w", podName, wErr)
		}

		go repository.SaveLog(ctx, line)
	}
	return nil
}
