package cmd

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/viper"

	labelerk8scli "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned"
)

// GetKubernetesClients returns all the required clients to communicate with
// Kubernetes: Labeler resources and core resources. CRDs are installed separately.
func GetKubernetesClients() (labelerk8scli.Interface, kubernetes.Interface, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(viper.GetString("master"), viper.GetString("kubeconfig"))
	if err != nil {
		return nil, nil, fmt.Errorf("could not load configuration: %w", err)
	}

	// Create clients.
	k8sCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	// App CRD k8s types client.
	nlCli, err := labelerk8scli.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	return nlCli, k8sCli, nil
}
