package labeler

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	"github.com/barpilot/node-labeler-operator/log"
	"github.com/spotahome/kooper/v2/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// LabelController watches nodes for one immutable copy of a Labeler rule.
type LabelController struct {
	l           *labelerv1alpha1.Labeler
	k8sCli      kubernetes.Interface
	logger      log.Logger
	mutex       sync.Mutex
	updateMutex sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
}

func NewLabelController(l *labelerv1alpha1.Labeler, k8sCli kubernetes.Interface, logger log.Logger) *LabelController {
	return &LabelController{l: l.DeepCopy(), k8sCli: k8sCli, logger: logger}
}

func (lc *LabelController) SameSpec(l *labelerv1alpha1.Labeler) bool {
	return lc.l.UID == l.UID && reflect.DeepEqual(lc.l.Spec, l.Spec)
}

func (lc *LabelController) Start() error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	if lc.cancel != nil {
		return fmt.Errorf("already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctrl, err := lc.controller(ctx)
	if err != nil {
		cancel()
		return err
	}
	lc.cancel = cancel
	lc.done = make(chan struct{})
	done := lc.done
	go func() {
		defer close(done)
		if err := ctrl.Run(ctx); err != nil && ctx.Err() == nil {
			lc.logger.Errorf("Labeler %s: %v", lc.l.Name, err)
		}
	}()
	return nil
}

// Stop cancels the watch and drains any active update before returning. Queued
// Kooper handlers also check this run's context, so they cannot restart writes.
func (lc *LabelController) Stop() error {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()
	if lc.cancel != nil {
		lc.cancel()
		<-lc.done
		lc.updateMutex.Lock()
		lc.updateMutex.Unlock()
		lc.cancel = nil
	}
	return nil
}

func (lc *LabelController) controller(ctx context.Context) (controller.Controller, error) {
	retr := controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return lc.k8sCli.CoreV1().Nodes().List(ctx, opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return lc.k8sCli.CoreV1().Nodes().Watch(ctx, opts)
		},
	})
	return controller.New(&controller.Config{
		Name:                 "labeler-" + lc.l.Name,
		ConcurrentWorkers:    1,
		ResyncInterval:       30 * time.Second,
		ProcessingJobRetries: 5,
		Retriever:            retr,
		Logger:               lc.logger,
		Handler:              controller.HandlerFunc(func(_ context.Context, obj runtime.Object) error { return lc.syncNode(ctx, obj) }),
	})
}

func (lc *LabelController) syncNode(ctx context.Context, obj runtime.Object) error {
	lc.updateMutex.Lock()
	defer lc.updateMutex.Unlock()
	if ctx.Err() != nil {
		return nil
	}
	node, ok := obj.(*corev1.Node)
	if !ok {
		return fmt.Errorf("invalid node object %T", obj)
	}
	if !NodeMatchesNodeSelectorTerms(node, lc.l.Spec.NodeSelectorTerms) {
		return nil
	}
	dst, err := mergeNode(node, lc.l.Spec.Merge)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(dst, node) {
		return nil
	}
	if lc.l.Spec.DryRun {
		lc.logger.Infof("dry run: Labeler %s would update Node %s", lc.l.Name, node.Name)
		return nil
	}
	// Keep the observed resourceVersion: conflicts go back to Kooper's queue,
	// which reads the latest informer object before retrying the merge.
	_, err = lc.k8sCli.CoreV1().Nodes().Update(ctx, dst, metav1.UpdateOptions{})
	if err == nil {
		lc.logger.Infof("Labeler %s updated Node %s", lc.l.Name, node.Name)
	}
	return err
}
