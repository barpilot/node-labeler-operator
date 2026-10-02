package labeler

import (
	"reflect"
	"slices"

	"dario.cat/mergo"
	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func mergeNode(node *corev1.Node, merge labelerv1alpha1.MergeSpec) (*corev1.Node, error) {
	dst := node.DeepCopy()
	src := merge.DeepCopy()
	// Keep existing map entries, including empty strings. General mergo merging
	// fills empty map values, which would change the original ownership rule.
	dst.Labels = addMapEntries(dst.Labels, src.Labels)
	dst.Annotations = addMapEntries(dst.Annotations, src.Annotations)
	src.Labels = nil
	src.Annotations = nil
	// Metadata slices were previously appended on every resync. Append each
	// requested entry only once, without dropping any existing entries.
	dst.Finalizers = appendUnique(dst.Finalizers, src.Finalizers)
	dst.OwnerReferences = appendUnique(dst.OwnerReferences, src.OwnerReferences)
	dst.ManagedFields = appendUnique(dst.ManagedFields, src.ManagedFields)
	src.Finalizers = nil
	src.OwnerReferences = nil
	src.ManagedFields = nil
	if err := mergo.Merge(&dst.ObjectMeta, src.ObjectMeta); err != nil {
		return nil, err
	}
	for _, taint := range src.Taints {
		if !slices.ContainsFunc(dst.Spec.Taints, func(existing corev1.Taint) bool { return existing.Key == taint.Key && existing.Effect == taint.Effect }) {
			dst.Spec.Taints = append(dst.Spec.Taints, taint)
		}
	}
	src.Taints = nil
	// Other exposed NodeSpec fields keep their original nonzero overwrite
	// semantics. They are subject to normal Kubernetes validation/admission.
	if err := mergo.Merge(&dst.Spec, src.NodeSpec, mergo.WithOverride); err != nil {
		return nil, err
	}
	if src.ConfigSource != nil {
		dst.Spec.ConfigSource = src.ConfigSource
	}
	if src.PodPreemptionPolicy != nil {
		dst.Spec.PodPreemptionPolicy = src.PodPreemptionPolicy
	}
	return dst, nil
}

func addMapEntries(dst, src map[string]string) map[string]string {
	for key, value := range src {
		if dst == nil {
			dst = make(map[string]string)
		}
		if _, exists := dst[key]; !exists {
			dst[key] = value
		}
	}
	return dst
}

func appendUnique[T any](dst, src []T) []T {
	// ponytail: quadratic in these small metadata lists; index entries if real workloads need it.
	for _, entry := range src {
		if !slices.ContainsFunc(dst, func(existing T) bool { return reflect.DeepEqual(existing, entry) }) {
			dst = append(dst, entry)
		}
	}
	return dst
}
