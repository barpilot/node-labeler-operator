package v1alpha1

import (
	v1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	scheme "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// LabelersGetter has a method to return a LabelerInterface.
// A group's client should implement this interface.
type LabelersGetter interface {
	Labelers() LabelerInterface
}

// LabelerInterface has methods to work with Labeler resources.
type LabelerInterface interface {
	Create(*v1alpha1.Labeler) (*v1alpha1.Labeler, error)
	Update(*v1alpha1.Labeler) (*v1alpha1.Labeler, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Labeler, error)
	List(opts v1.ListOptions) (*v1alpha1.LabelerList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Labeler, err error)
	LabelerExpansion
}

// labelers implements LabelerInterface
type labelers struct {
	client rest.Interface
}

// newLabelers returns a Labelers
func newLabelers(c *LabelerV1alpha1Client) *labelers {
	return &labelers{
		client: c.RESTClient(),
	}
}

// Get takes name of the labeler, and returns the corresponding labeler object, and an error if there is any.
func (c *labelers) Get(name string, options v1.GetOptions) (result *v1alpha1.Labeler, err error) {
	result = &v1alpha1.Labeler{}
	err = c.client.Get().
		Resource("labelers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Labelers that match those selectors.
func (c *labelers) List(opts v1.ListOptions) (result *v1alpha1.LabelerList, err error) {
	result = &v1alpha1.LabelerList{}
	err = c.client.Get().
		Resource("labelers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested labelers.
func (c *labelers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("labelers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a labeler and creates it.  Returns the server's representation of the labeler, and an error, if there is any.
func (c *labelers) Create(labeler *v1alpha1.Labeler) (result *v1alpha1.Labeler, err error) {
	result = &v1alpha1.Labeler{}
	err = c.client.Post().
		Resource("labelers").
		Body(labeler).
		Do().
		Into(result)
	return
}

// Update takes the representation of a labeler and updates it. Returns the server's representation of the labeler, and an error, if there is any.
func (c *labelers) Update(labeler *v1alpha1.Labeler) (result *v1alpha1.Labeler, err error) {
	result = &v1alpha1.Labeler{}
	err = c.client.Put().
		Resource("labelers").
		Name(labeler.Name).
		Body(labeler).
		Do().
		Into(result)
	return
}

// Delete takes name of the labeler and deletes it. Returns an error if one occurs.
func (c *labelers) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("labelers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *labelers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("labelers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched labeler.
func (c *labelers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Labeler, err error) {
	result = &v1alpha1.Labeler{}
	err = c.client.Patch(pt).
		Resource("labelers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
