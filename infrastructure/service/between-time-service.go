package service

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/repository"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func BetweenTimesProcess(ctx context.Context, cfg *domain.Config, startTime, endTime time.Time) {
	clientset, err := GetKubernetesClient()
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
			go repository.SaveLog(ctx, line)
		}
	}
}

func getPodsByLabel(clientset *kubernetes.Clientset, cfg *domain.Config) ([]corev1.Pod, error) {
	podList, err := clientset.CoreV1().Pods(cfg.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: cfg.LabelSelector,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func extractTimestamp(logLine string) (time.Time, error) {
	for _, r := range domain.TimeRegexes {
		if match := r.FindStringSubmatch(logLine); match != nil {
			return parseTimestamp(match[1])
		} else {
			log.Printf("No se pudo parsear la línea: %s", logLine)
		}
	}
	return time.Time{}, fmt.Errorf("no timestamp found")
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
