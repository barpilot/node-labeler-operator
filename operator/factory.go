package operator

import (
	"context"
	"fmt"

	"github.com/spotahome/kooper/v2/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	labelerk8scli "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned"
	"github.com/barpilot/node-labeler-operator/log"
	"github.com/barpilot/node-labeler-operator/service/labeler"
)

type operator struct {
	cfg     Config
	client  labelerk8scli.Interface
	service *labeler.Labeler
	logger  log.Logger
}

// New watches Labeler rules and runs a Kooper node controller for each rule.
func New(cfg Config, client labelerk8scli.Interface, kubeCli kubernetes.Interface, logger log.Logger) (controller.Controller, error) {
	if cfg.ResyncPeriod <= 0 {
		return nil, fmt.Errorf("resync period must be positive")
	}
	return &operator{cfg: cfg, client: client, service: labeler.NewLabeler(kubeCli, logger), logger: logger}, nil
}

func (o *operator) Run(ctx context.Context) error {
	// Kooper v2 ignores deleted objects. A native informer keeps rule deletion
	// (including deletion discovered on relist) connected to controller shutdown.
	informer := cache.NewSharedIndexInformer(&cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			return o.client.LabelerV1alpha1().Labelers().List(ctx, opts)
		},
		WatchFuncWithContext: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			return o.client.LabelerV1alpha1().Labelers().Watch(ctx, opts)
		},
	}, &labelerv1alpha1.Labeler{}, o.cfg.ResyncPeriod, cache.Indexers{})
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    o.ensure,
		UpdateFunc: func(_, obj interface{}) { o.ensure(obj) },
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err != nil {
				o.logger.Errorf("invalid deleted Labeler: %v", err)
				return
			}
			if err := o.service.DeleteLabeler(key); err != nil {
				o.logger.Errorf("stop Labeler %s: %v", key, err)
			}
		},
	})
	if err != nil {
		return err
	}
	defer o.service.Stop()
	informer.RunWithContext(ctx)
	return nil
}

func (o *operator) ensure(obj interface{}) {
	l, ok := obj.(*labelerv1alpha1.Labeler)
	if !ok {
		o.logger.Errorf("invalid Labeler object %T", obj)
		return
	}
	if err := o.service.EnsureLabeler(l); err != nil {
		o.logger.Errorf("start Labeler %s: %v", l.Name, err)
	}
}
