package labeler

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	"github.com/spotahome/kooper/v2/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientfeatures "k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func matchingRule() *labelerv1alpha1.Labeler {
	return &labelerv1alpha1.Labeler{ObjectMeta: metav1.ObjectMeta{Name: "example"}, Spec: labelerv1alpha1.LabelerSpec{
		NodeSelector: corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "pool", Operator: corev1.NodeSelectorOpIn, Values: []string{"workers"}}}}}},
	}}
}

func TestMergePreservesOwnershipAndIsIdempotent(t *testing.T) {
	original := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node", ResourceVersion: "7", Labels: map[string]string{"pool": "workers", "existing": "", "other": "keep"}, Annotations: map[string]string{"existing": "owner"}, Finalizers: []string{"example.org/existing"}}, Spec: corev1.NodeSpec{Taints: []corev1.Taint{
		{Key: "node.kubernetes.io/not-ready", Effect: corev1.TaintEffectNoSchedule},
		{Key: "dedicated", Value: "existing", Effect: corev1.TaintEffectNoSchedule},
	}}, Status: corev1.NodeStatus{Phase: corev1.NodeRunning}}
	before := original.DeepCopy()
	rule := matchingRule()
	rule.Spec.Merge = labelerv1alpha1.MergeSpec{ObjectMeta: metav1.ObjectMeta{
		Labels: map[string]string{"existing": "do-not-overwrite", "new": "added"}, Annotations: map[string]string{"existing": "do-not-overwrite", "new": "added"},
		Finalizers:      []string{"example.org/existing", "example.org/new", "example.org/new"},
		OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "Node", Name: "owner", UID: "owner-uid"}},
	}, NodeSpec: corev1.NodeSpec{Taints: []corev1.Taint{
		{Key: "dedicated", Value: "requested", Effect: corev1.TaintEffectNoSchedule},
		{Key: "dedicated", Value: "new-effect", Effect: corev1.TaintEffectPreferNoSchedule},
		{Key: "dedicated", Value: "duplicate", Effect: corev1.TaintEffectPreferNoSchedule},
	}}}
	client := fake.NewClientset(original)
	lc := NewLabelController(rule, client, log.Dummy)
	if err := lc.syncNode(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	actions := client.Actions()
	if len(actions) != 1 || actions[0].GetVerb() != "update" || actions[0].GetSubresource() != "" {
		t.Fatalf("unexpected actions: %v", actions)
	}
	got := actions[0].(ktesting.UpdateAction).GetObject().(*corev1.Node)
	if got.Labels["existing"] != "" || got.Labels["new"] != "added" || got.Labels["other"] != "keep" || got.Annotations["existing"] != "owner" {
		t.Fatalf("metadata ownership changed: %v", got.ObjectMeta)
	}
	if len(got.Spec.Taints) != 3 || !reflect.DeepEqual(got.Spec.Taints[:2], original.Spec.Taints) || got.Spec.Taints[2].Value != "new-effect" {
		t.Fatalf("taints changed ownership: %v", got.Spec.Taints)
	}
	if got.ResourceVersion != "7" || !reflect.DeepEqual(got.Status, original.Status) {
		t.Fatal("resource version or status changed")
	}
	if len(got.Finalizers) != 2 || len(got.OwnerReferences) != 1 {
		t.Fatalf("metadata entries duplicated: %+v", got.ObjectMeta)
	}
	if !reflect.DeepEqual(original, before) {
		t.Fatal("mutated informer object")
	}
	if err := lc.syncNode(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if len(client.Actions()) != 1 {
		t.Fatal("second sync should not write")
	}
}

func TestOverlappingRulesAreFirstWins(t *testing.T) {
	for _, values := range [][]string{{"a", "b"}, {"b", "a"}} {
		node := &corev1.Node{}
		for _, value := range values {
			var err error
			node, err = mergeNode(node, labelerv1alpha1.MergeSpec{NodeSpec: corev1.NodeSpec{Taints: []corev1.Taint{{Key: "dedicated", Effect: corev1.TaintEffectNoSchedule, Value: value}}}})
			if err != nil {
				t.Fatal(err)
			}
		}
		if len(node.Spec.Taints) != 1 || node.Spec.Taints[0].Value != values[0] {
			t.Fatalf("first value must win for order %v: %v", values, node.Spec.Taints)
		}
	}
}

func TestOtherExposedFieldsStillMerge(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node", GenerateName: "original-"}, Spec: corev1.NodeSpec{PodCIDR: "10.1.0.0/24", ProviderID: "old", Unschedulable: true, ConfigSource: &corev1.NodeConfigSource{ConfigMap: &corev1.ConfigMapNodeConfigSource{Name: "old", Namespace: "old"}}}}
	rule := labelerv1alpha1.MergeSpec{ObjectMeta: metav1.ObjectMeta{GenerateName: "replacement-", Finalizers: []string{"example.org/protection"}}, NodeSpec: corev1.NodeSpec{PodCIDR: "10.2.0.0/24", PodCIDRs: []string{"10.2.0.0/24"}, ProviderID: "new", ConfigSource: &corev1.NodeConfigSource{ConfigMap: &corev1.ConfigMapNodeConfigSource{Name: "new"}}}}
	got, err := mergeNode(node, rule)
	if err != nil {
		t.Fatal(err)
	}
	if got.GenerateName != "original-" || got.Finalizers[0] != "example.org/protection" || got.Spec.PodCIDR != rule.PodCIDR || !reflect.DeepEqual(got.Spec.PodCIDRs, rule.PodCIDRs) || got.Spec.ProviderID != "new" || !got.Spec.Unschedulable || !reflect.DeepEqual(got.Spec.ConfigSource, rule.ConfigSource) {
		t.Fatalf("legacy merge fields changed: %+v", got)
	}
}

func TestNoWriteForDryRunUnmatchedAndCanceled(t *testing.T) {
	for _, mode := range []string{"dry-run", "unmatched", "empty-selector", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			rule := matchingRule()
			rule.Spec.Merge.Labels = map[string]string{"add": "value"}
			node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node", Labels: map[string]string{"pool": "workers"}}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "dry-run":
				rule.Spec.DryRun = true
			case "unmatched":
				node.Labels["pool"] = "other"
			case "empty-selector":
				rule.Spec.NodeSelectorTerms = nil
			case "canceled":
				cancel()
			}
			client := fake.NewClientset(node)
			if err := NewLabelController(rule, client, log.Dummy).syncNode(ctx, node); err != nil {
				t.Fatal(err)
			}
			if len(client.Actions()) != 0 {
				t.Fatalf("unexpected API calls: %v", client.Actions())
			}
		})
	}
}

func TestErrorsReachControllerQueue(t *testing.T) {
	rule := matchingRule()
	rule.Spec.Merge.Labels = map[string]string{"add": "value"}
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node", Labels: map[string]string{"pool": "workers"}}}
	client := fake.NewClientset(node)
	failure := apierrors.NewConflict(schema.GroupResource{Resource: "nodes"}, "node", errors.New("changed"))
	client.PrependReactor("update", "nodes", func(ktesting.Action) (bool, runtime.Object, error) { return true, nil, failure })
	if err := NewLabelController(rule, client, log.Dummy).syncNode(context.Background(), node); !apierrors.IsConflict(err) {
		t.Fatalf("lost conflict: %v", err)
	}
}

func TestConflictRetryPreservesConcurrentChanges(t *testing.T) {
	// Fake watches cannot emit the streaming-list initialization bookmark.
	featuretesting.SetFeatureDuringTest(t, clientfeatures.WatchListClient, false)
	rule := matchingRule()
	rule.Spec.Merge.Labels = map[string]string{"add": "value"}
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node", ResourceVersion: "1", Labels: map[string]string{"pool": "workers"}}}
	client := fake.NewClientset(node)
	var attempts atomic.Int32
	client.PrependReactor("update", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		requested := action.(ktesting.UpdateAction).GetObject().(*corev1.Node)
		if attempts.Add(1) == 1 {
			concurrent := node.DeepCopy()
			concurrent.ResourceVersion = "2"
			concurrent.Labels["concurrent"] = "keep"
			concurrent.Spec.Taints = []corev1.Taint{{Key: "node.kubernetes.io/not-ready", Effect: corev1.TaintEffectNoSchedule}}
			if err := client.Tracker().Update(corev1.SchemeGroupVersion.WithResource("nodes"), concurrent, ""); err != nil {
				return true, nil, err
			}
		}
		if requested.ResourceVersion != "2" {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "nodes"}, node.Name, errors.New("stale"))
		}
		return false, nil, nil
	})
	lc := NewLabelController(rule, client, log.Dummy)
	if err := lc.Start(); err != nil {
		t.Fatal(err)
	}
	defer lc.Stop()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := client.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if got.Labels["add"] == "value" {
			if attempts.Load() < 2 || got.Labels["concurrent"] != "keep" || len(got.Spec.Taints) != 1 {
				t.Fatalf("lost concurrent change: %+v", got)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("controller did not retry the conflict")
}
