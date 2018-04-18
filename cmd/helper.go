package cmd

import (
	"fmt"

	apiextensionscli "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/viper"
	"github.com/spotahome/kooper/client/crd"

	labelerk8scli "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned"
	"github.com/barpilot/node-labeler-operator/log"
)

// GetKubernetesClients returns all the required clients to communicate with
// kubernetes cluster: CRD type client, pod terminator types client, kubernetes core types client.
func GetKubernetesClients(logger log.Logger) (labelerk8scli.Interface, crd.Interface, kubernetes.Interface, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(viper.GetString("master"), viper.GetString("kubeconfig"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not load configuration: %s", err)
	}

	// Create clients.
	k8sCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	// App CRD k8s types client.
	nlCli, err := labelerk8scli.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	// CRD cli.
	aexCli, err := apiextensionscli.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	crdCli := crd.NewClient(aexCli, logger)

	return nlCli, crdCli, k8sCli, nil
}
