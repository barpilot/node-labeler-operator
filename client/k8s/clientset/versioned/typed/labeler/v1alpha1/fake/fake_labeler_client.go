package fake

import (
	v1alpha1 "github.com/barpilot/node-labeler-operator/client/k8s/clientset/versioned/typed/labeler/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeLabelerV1alpha1 struct {
	*testing.Fake
}

func (c *FakeLabelerV1alpha1) Labelers() v1alpha1.LabelerInterface {
	return &FakeLabelers{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeLabelerV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
