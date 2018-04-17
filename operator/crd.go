package operator

import (
	"github.com/spotahome/kooper/client/crd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	labelv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	labelerk8scli "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned"
)

// LabelerCRD is the crd pod terminator.
type LabelerCRD struct {
	crdCli     crd.Interface
	kubecCli   kubernetes.Interface
	labelerCli labelerk8scli.Interface
}

func newLabelerCRD(labelerCli labelerk8scli.Interface, crdCli crd.Interface, kubeCli kubernetes.Interface) *LabelerCRD {
	return &LabelerCRD{
		crdCli:     crdCli,
		labelerCli: labelerCli,
		kubecCli:   kubeCli,
	}
}

// LabelerCRD satisfies resource.crd interface.
func (p *LabelerCRD) Initialize() error {
	crd := crd.Conf{
		Kind:       labelv1alpha1.LabelerKind,
		NamePlural: labelv1alpha1.LabelerNamePlural,
		Group:      labelv1alpha1.SchemeGroupVersion.Group,
		Version:    labelv1alpha1.SchemeGroupVersion.Version,
		Scope:      labelv1alpha1.LabelerScope,
	}

	return p.crdCli.EnsurePresent(crd)
}

// GetListerWatcher satisfies resource.crd interface (and retrieve.Retriever).
func (p *LabelerCRD) GetListerWatcher() cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return p.labelerCli.LabelerV1alpha1().Labelers().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return p.labelerCli.LabelerV1alpha1().Labelers().Watch(options)
		},
	}
}

// GetObject satisfies resource.crd interface (and retrieve.Retriever).
func (p *LabelerCRD) GetObject() runtime.Object {
	return &labelv1alpha1.Labeler{}
}
