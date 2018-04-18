package fake

import (
	v1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeLabelers implements LabelerInterface
type FakeLabelers struct {
	Fake *FakeLabelerV1alpha1
}

var labelersResource = schema.GroupVersionResource{Group: "labeler.barpilot.io", Version: "v1alpha1", Resource: "labelers"}

var labelersKind = schema.GroupVersionKind{Group: "labeler.barpilot.io", Version: "v1alpha1", Kind: "Labeler"}

// Get takes name of the labeler, and returns the corresponding labeler object, and an error if there is any.
func (c *FakeLabelers) Get(name string, options v1.GetOptions) (result *v1alpha1.Labeler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(labelersResource, name), &v1alpha1.Labeler{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Labeler), err
}

// List takes label and field selectors, and returns the list of Labelers that match those selectors.
func (c *FakeLabelers) List(opts v1.ListOptions) (result *v1alpha1.LabelerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(labelersResource, labelersKind, opts), &v1alpha1.LabelerList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.LabelerList{}
	for _, item := range obj.(*v1alpha1.LabelerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested labelers.
func (c *FakeLabelers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(labelersResource, opts))
}

// Create takes the representation of a labeler and creates it.  Returns the server's representation of the labeler, and an error, if there is any.
func (c *FakeLabelers) Create(labeler *v1alpha1.Labeler) (result *v1alpha1.Labeler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(labelersResource, labeler), &v1alpha1.Labeler{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Labeler), err
}

// Update takes the representation of a labeler and updates it. Returns the server's representation of the labeler, and an error, if there is any.
func (c *FakeLabelers) Update(labeler *v1alpha1.Labeler) (result *v1alpha1.Labeler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(labelersResource, labeler), &v1alpha1.Labeler{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Labeler), err
}

// Delete takes name of the labeler and deletes it. Returns an error if one occurs.
func (c *FakeLabelers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(labelersResource, name), &v1alpha1.Labeler{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeLabelers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(labelersResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.LabelerList{})
	return err
}

// Patch applies the patch and returns the patched labeler.
func (c *FakeLabelers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Labeler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(labelersResource, name, data, subresources...), &v1alpha1.Labeler{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Labeler), err
}
