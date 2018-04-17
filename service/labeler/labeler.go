package labeler

import (
	"sync"

	"k8s.io/client-go/kubernetes"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	"github.com/barpilot/node-labeler-operator/log"
)

// Syncer is the interface that every labeler service implementation
// needs to implement.
type Syncer interface {
	// EnsureLabeler will ensure that the labeler is running and working.
	EnsureLabeler(pt *labelerv1alpha1.Labeler) error
	// DeleteLabeler will stop and delete the labeler.
	DeleteLabeler(name string) error
}

// Chaos is the service that will ensure that the desired pod terminator CRDs are met.
// Chaos will have running instances of PodDestroyers.
type Labeler struct {
	k8sCli kubernetes.Interface
	reg    sync.Map
	logger log.Logger
}

// NewChaos returns a new Chaos service.
func NewLabeler(k8sCli kubernetes.Interface, logger log.Logger) *Labeler {
	return &Labeler{
		k8sCli: k8sCli,
		reg:    sync.Map{},
		logger: logger,
	}
}

// EnsurePodTerminator satisfies ChaosSyncer interface.
func (c *Labeler) EnsureLabeler(l *labelerv1alpha1.Labeler) error {
	labelController, ok := c.reg.Load(l.Name)
	var lc *LabelController

	// We are already running.
	if ok {
		lc = labelController.(*LabelController)
		// If not the same spec means options have changed, so we don't longer need this pod killer.
		if !lc.SameSpec(l) {
			c.logger.Infof("spec of %s changed, recreating label controller", l.Name)
			if err := c.DeleteLabeler(l.Name); err != nil {
				return err
			}
		} else { // We are ok, nothing changed.
			return nil
		}
	}

	// Create a pod killer.
	lCopy := l.DeepCopy()
	lc = NewLabelController(lCopy, c.k8sCli, c.logger)
	c.reg.Store(l.Name, lc)
	return lc.Start()
	// TODO: garbage collection.
}

// DeletePodTerminator satisfies ChaosSyncer interface.
func (c *Labeler) DeleteLabeler(name string) error {
	l, ok := c.reg.Load(name)
	if !ok {
		return nil
	}

	lc := l.(*LabelController)
	if err := lc.Stop(); err != nil {
		return err
	}

	c.reg.Delete(name)
	return nil
}
