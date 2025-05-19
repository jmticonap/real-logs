package service

import (
	"fmt"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func GetKubernetesClient() (*kubernetes.Clientset, error) {
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
