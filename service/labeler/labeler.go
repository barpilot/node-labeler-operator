package labeler

import (
	"sync"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	"github.com/barpilot/node-labeler-operator/log"
	"k8s.io/client-go/kubernetes"
)

// Labeler owns the running node controllers. Changes and deletion stop the old
// controller but never undo attributes that have already been applied.
type Labeler struct {
	k8sCli  kubernetes.Interface
	logger  log.Logger
	mutex   sync.Mutex
	reg     map[string]*LabelController
	stopped bool
}

func NewLabeler(k8sCli kubernetes.Interface, logger log.Logger) *Labeler {
	return &Labeler{k8sCli: k8sCli, logger: logger, reg: make(map[string]*LabelController)}
}

func (c *Labeler) EnsureLabeler(l *labelerv1alpha1.Labeler) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.stopped {
		return nil
	}
	if lc := c.reg[l.Name]; lc != nil {
		if lc.SameSpec(l) {
			return nil
		}
		if err := lc.Stop(); err != nil {
			return err
		}
		delete(c.reg, l.Name)
	}
	lc := NewLabelController(l, c.k8sCli, c.logger)
	if err := lc.Start(); err != nil {
		return err
	}
	c.reg[l.Name] = lc
	return nil
}

func (c *Labeler) DeleteLabeler(name string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if lc := c.reg[name]; lc != nil {
		if err := lc.Stop(); err != nil {
			return err
		}
		delete(c.reg, name)
	}
	return nil
}

// Stop closes the registry, including callbacks already queued by the informer.
func (c *Labeler) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.stopped = true
	for name, lc := range c.reg {
		lc.Stop()
		delete(c.reg, name)
	}
}
