package service

import (
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetKubernetesClient returns a Kubernetes clientset with a default timeout of 60 seconds.
func GetKubernetesClient() (*kubernetes.Clientset, error) {
	return GetKubernetesClientWithTimeout(60 * time.Second)
}

// GetKubernetesClientWithTimeout returns a Kubernetes clientset with a specified timeout.
// If timeout is 0, no timeout is set.
func GetKubernetesClientWithTimeout(timeout time.Duration) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	// Intenta config InCluster
	config, err = rest.InClusterConfig()
	if err != nil {
		// Si falla, intenta con kubeconfig local
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("error creando config Kubernetes: %w", err)
		}
	}

	// Aumentar el timeout si es mayor que 0
	if timeout > 0 {
		config.Timeout = timeout
	}

	return kubernetes.NewForConfig(config)
}
