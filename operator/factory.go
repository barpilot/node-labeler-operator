package operator

import (
	"github.com/spotahome/kooper/client/crd"
	"github.com/spotahome/kooper/operator"
	"github.com/spotahome/kooper/operator/controller"
	"k8s.io/client-go/kubernetes"

	labelerk8scli "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned"
	"github.com/barpilot/node-labeler-operator/log"
)

// New returns pod terminator operator.
func New(cfg Config, labelerCli labelerk8scli.Interface, crdCli crd.Interface, kubeCli kubernetes.Interface, logger log.Logger) (operator.Operator, error) {

	// Create crd.
	ptCRD := newLabelerCRD(labelerCli, crdCli, kubeCli)

	// Create handler.
	handler := newHandler(kubeCli, logger)

	// Create controller.
	ctrl := controller.NewSequential(cfg.ResyncPeriod, handler, ptCRD, nil, logger)

	// Assemble CRD and controller to create the operator.
	return operator.NewOperator(ptCRD, ctrl, logger), nil
}
