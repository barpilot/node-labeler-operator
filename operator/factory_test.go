package operator

import (
	"context"
	"testing"
	"time"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	labelfake "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned/fake"
	"github.com/spotahome/kooper/v2/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	clientfeatures "k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestRuleEditDeleteAndShutdownStopEnforcement(t *testing.T) {
	// Fake watches cannot emit the streaming-list initialization bookmark.
	featuretesting.SetFeatureDuringTest(t, clientfeatures.WatchListClient, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rule := &labelerv1alpha1.Labeler{ObjectMeta: metav1.ObjectMeta{Name: "example", UID: "rule-1"}, Spec: labelerv1alpha1.LabelerSpec{
		NodeSelector: corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "pool", Operator: corev1.NodeSelectorOpExists}}}}},
		Merge:        labelerv1alpha1.MergeSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"old-rule": "applied"}}},
	}}
	labelClient := labelfake.NewSimpleClientset(rule)
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node", Labels: map[string]string{"pool": "workers"}}}
	kubeClient := fake.NewClientset(node)
	watches := make(chan *watch.RaceFreeFakeWatcher, 8)
	kubeClient.PrependWatchReactor("nodes", func(ktesting.Action) (bool, watch.Interface, error) {
		w := watch.NewRaceFreeFake()
		watches <- w
		return true, w, nil
	})
	op, err := New(NewOperatorConfig(time.Second), labelClient, kubeClient, log.Dummy)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- op.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("operator did not stop")
		}
	}()
	nextWatch := func() *watch.RaceFreeFakeWatcher {
		t.Helper()
		select {
		case w := <-watches:
			return w
		case <-time.After(5 * time.Second):
			t.Fatal("node watch did not start")
			return nil
		}
	}
	await := func(message string, condition func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if condition() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal(message)
	}
	first := nextWatch()
	await("initial rule did not apply", func() bool {
		n, _ := kubeClient.CoreV1().Nodes().Get(ctx, "node", metav1.GetOptions{})
		return n.Labels["old-rule"] == "applied"
	})
	rule.Spec.Merge.Labels = map[string]string{"new-rule": "applied"}
	if _, err := labelClient.LabelerV1alpha1().Labelers().Update(ctx, rule, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	second := nextWatch()
	await("edit left old controller watching nodes", first.IsStopped)
	await("edited rule did not apply", func() bool {
		n, _ := kubeClient.CoreV1().Nodes().Get(ctx, "node", metav1.GetOptions{})
		return n.Labels["new-rule"] == "applied" && n.Labels["old-rule"] == "applied"
	})
	// A node appearing after the edit must receive only the new rule.
	replacement := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "replacement", Labels: map[string]string{"pool": "workers"}}}
	if _, err := kubeClient.CoreV1().Nodes().Create(ctx, replacement, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	second.Add(replacement)
	await("edited controller did not label replacement", func() bool {
		n, _ := kubeClient.CoreV1().Nodes().Get(ctx, "replacement", metav1.GetOptions{})
		return n.Labels["new-rule"] == "applied" && n.Labels["old-rule"] == ""
	})
	if err := labelClient.LabelerV1alpha1().Labelers().Delete(ctx, rule.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	await("deletion left controller running", second.IsStopped)
	n, err := kubeClient.CoreV1().Nodes().Get(ctx, "node", metav1.GetOptions{})
	if err != nil || n.Labels["old-rule"] != "applied" || n.Labels["new-rule"] != "applied" {
		t.Fatalf("deletion removed applied attributes: %v %v", n, err)
	}
	rule.UID = "rule-2"
	if _, err := labelClient.LabelerV1alpha1().Labelers().Create(ctx, rule, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	third := nextWatch()
	cancel()
	await("operator shutdown left controller running", third.IsStopped)
	for _, action := range kubeClient.Actions() {
		if action.GetSubresource() != "" {
			t.Fatalf("unexpected subresource action: %v", action)
		}
	}
}
