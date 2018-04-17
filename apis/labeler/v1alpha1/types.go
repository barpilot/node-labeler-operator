package v1alpha1

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodTerminator represents a pod terminator.
type Labeler struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the ddesired behaviour of the pod terminator.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec LabelerSpec `json:"spec,omitempty"`
}

// LabelerSpec is the spec for a Labeler resource.
type LabelerSpec struct {
	// Selector is how the target will be selected.
	v1.NodeSelector `json:",inline"`

	// Size is how many nodes to label.
	//Size int `json:"Size,omitempty"`
	// TerminationPercent is the percent of pods that will be killed randomly.
	Merge MergeSpec `json:"merge,omitempty"`
	// DryRun will set the killing in dryrun mode or not.
	// +optional
	DryRun bool `json:"dryRun,omitempty"`
}

type MergeSpec struct {
	metav1.ObjectMeta `json:",inline" protobuf:"bytes,1,opt,name=metadata"`

	v1.NodeSpec `json:",inline" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LabelerList is a list of Labeler resources
type LabelerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Labeler `json:"items"`
}
