package operator

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	"github.com/barpilot/node-labeler-operator/log"
	"github.com/barpilot/node-labeler-operator/service/labeler"
)

// Handler  is the pod terminator handler that will handle the
// events received from kubernetes.
type handler struct {
	labelerService labeler.Syncer
	logger         log.Logger
}

// newHandler returns a new handler.
func newHandler(k8sCli kubernetes.Interface, logger log.Logger) *handler {
	return &handler{
		labelerService: labeler.NewLabeler(k8sCli, logger),
		logger:         logger,
	}
}

// Add will ensure that the required pod terminator is running.
func (h *handler) Add(obj runtime.Object) error {
	l, ok := obj.(*labelerv1alpha1.Labeler)
	if !ok {
		return fmt.Errorf("%v is not a labeler object", obj.GetObjectKind())
	}

	return h.labelerService.EnsureLabeler(l)
}

// Delete will ensure the reuited pod terminator is not running.
func (h *handler) Delete(name string) error {
	return h.labelerService.DeleteLabeler(name)
}
