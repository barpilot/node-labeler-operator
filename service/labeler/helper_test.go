package labeler

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestNodeSelectors(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"pool": "workers", "count": "3"}}}
	for _, tc := range []struct {
		name string
		req  corev1.NodeSelectorRequirement
		want bool
	}{
		{"in", corev1.NodeSelectorRequirement{Key: "pool", Operator: corev1.NodeSelectorOpIn, Values: []string{"workers"}}, true},
		{"not-in", corev1.NodeSelectorRequirement{Key: "pool", Operator: corev1.NodeSelectorOpNotIn, Values: []string{"control"}}, true},
		{"exists", corev1.NodeSelectorRequirement{Key: "pool", Operator: corev1.NodeSelectorOpExists}, true},
		{"absent", corev1.NodeSelectorRequirement{Key: "other", Operator: corev1.NodeSelectorOpDoesNotExist}, true},
		{"greater", corev1.NodeSelectorRequirement{Key: "count", Operator: corev1.NodeSelectorOpGt, Values: []string{"2"}}, true},
		{"less", corev1.NodeSelectorRequirement{Key: "count", Operator: corev1.NodeSelectorOpLt, Values: []string{"2"}}, false},
		{"invalid", corev1.NodeSelectorRequirement{Key: "count", Operator: "invalid"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NodeMatchesNodeSelectorTerms(node, []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{tc.req}}}); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	if NodeMatchesNodeSelectorTerms(node, []corev1.NodeSelectorTerm{{}}) {
		t.Fatal("empty term must not match")
	}
	terms := []corev1.NodeSelectorTerm{
		{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "pool", Operator: corev1.NodeSelectorOpIn, Values: []string{"other"}}}},
		{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "count", Operator: corev1.NodeSelectorOpExists}, {Key: "pool", Operator: corev1.NodeSelectorOpExists}}},
	}
	if !NodeMatchesNodeSelectorTerms(node, terms) {
		t.Fatal("terms must be ORed and expressions ANDed")
	}
}
