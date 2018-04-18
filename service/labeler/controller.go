package labeler

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	"github.com/barpilot/node-labeler-operator/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/spotahome/kooper/operator/controller"
	"github.com/spotahome/kooper/operator/handler"
	"github.com/spotahome/kooper/operator/retrieve"

	"github.com/imdario/mergo"
)

// PodKiller will kill pods at regular intervals.
type LabelController struct {
	l      *labelerv1alpha1.Labeler
	k8sCli kubernetes.Interface
	logger log.Logger

	running bool
	mutex   sync.Mutex
	stopC   chan struct{}
}

// NewCustomPodKiller is a constructor that lets you customize everything on the object construction.
func NewLabelController(l *labelerv1alpha1.Labeler, k8sCli kubernetes.Interface, logger log.Logger) *LabelController {
	return &LabelController{
		l:      l,
		k8sCli: k8sCli,
		logger: logger,
	}
}

// SameSpec checks if the label controller has the same spec.
func (lc *LabelController) SameSpec(l *labelerv1alpha1.Labeler) bool {
	return reflect.DeepEqual(lc.l.Spec, l.Spec)
}

// Start will run the pod killer at regular intervals.
func (lc *LabelController) Start() error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	if lc.running {
		return fmt.Errorf("already running")
	}

	lc.stopC = make(chan struct{})
	lc.running = true

	go func() {
		lc.logger.Infof("started %s label controller", lc.l.Name)
		if err := lc.run(); err != nil {
			lc.logger.Errorf("error executing label controller: %s", err)
		}
	}()

	return nil
}

// Stop stops the pod killer.
func (lc *LabelController) Stop() error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	if lc.running {
		close(lc.stopC)
		lc.logger.Infof("stopped %s label controller", lc.l.Name)
	}

	lc.running = false
	return nil
}

// run will run the loop that will kill eventually the required pods.
func (lc *LabelController) run() error {
	// Create our retriever so the controller knows how to get/listen for pod events.
	retr := &retrieve.Resource{
		Object: &corev1.Node{},
		ListerWatcher: &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return lc.k8sCli.CoreV1().Nodes().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return lc.k8sCli.CoreV1().Nodes().Watch(options)
			},
		},
	}

	// Our domain logic that will print every add/sync/update and delete event we .
	hand := &handler.HandlerFunc{
		AddFunc: func(obj runtime.Object) error {
			node, ok := obj.(*corev1.Node)
			if !ok {
				return fmt.Errorf("invalid node object %s", node)
			}
			lc.logger.Infof("Node updated: %s", node.Name)

			if !NodeMatchesNodeSelectorTerms(node, lc.l.Spec.NodeSelectorTerms) {
				lc.logger.Infof("Node unmatch")
				return nil
			}

			//dst := *lc.l.Spec.Merge.DeepCopy()
			dst := node.DeepCopy()

			if err := mergo.Merge(&dst.ObjectMeta, lc.l.Spec.Merge.ObjectMeta, mergo.WithAppendSlice); err != nil {
				lc.logger.Infof("merge error: %v", err)
			}

			if err := mergo.Merge(&dst.Spec, lc.l.Spec.Merge.NodeSpec, mergo.WithAppendSlice); err != nil {
				lc.logger.Infof("merge error: %v", err)
			}

			if reflect.DeepEqual(dst, node) {
				lc.logger.Infof("Node unchanged")
				return nil
			}
			_, err := lc.k8sCli.CoreV1().Nodes().Update(dst)
			lc.logger.Infof("Node updated")
			return err
		},
		DeleteFunc: func(s string) error {
			// log.Infof("Node deleted: %s", s)
			return nil
		},
	}

	// Create the controller that will refresh every 30 seconds.

	ctrl := controller.NewSequential(30*time.Second, hand, retr, nil, lc.logger)
	// Start our controller.
	stopC := make(chan struct{})
	return ctrl.Run(stopC)
}
