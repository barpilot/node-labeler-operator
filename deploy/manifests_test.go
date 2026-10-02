package deploy

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	labelerv1alpha1 "github.com/barpilot/node-labeler-operator/apis/labeler/v1alpha1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdvalidation "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

func TestCRDAndExample(t *testing.T) {
	data, err := os.ReadFile("crd.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var crd apiextensionsv1.CustomResourceDefinition
	if err := yaml.UnmarshalStrict(data, &crd); err != nil {
		t.Fatal(err)
	}
	if crd.APIVersion != "apiextensions.k8s.io/v1" || crd.Spec.Group != labelerv1alpha1.SchemeGroupVersion.Group || crd.Spec.Scope != "Cluster" || crd.Spec.Names.Kind != labelerv1alpha1.LabelerKind {
		t.Fatal("CRD identity drift")
	}
	if len(crd.Spec.Versions) != 1 || crd.Spec.Versions[0].Name != labelerv1alpha1.SchemeGroupVersion.Version {
		t.Fatal("CRD version drift")
	}
	// Run the same defaulting and validation used by the Kubernetes API server.
	apiextensionsv1.SetDefaults_CustomResourceDefinition(&crd)
	var internal apiextensions.CustomResourceDefinition
	if err := apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(&crd, &internal, nil); err != nil {
		t.Fatal(err)
	}
	if errs := crdvalidation.ValidateCustomResourceDefinition(context.Background(), &internal); len(errs) != 0 {
		t.Fatalf("invalid CRD: %v", errs)
	}
	schema := internal.Spec.Validation.OpenAPIV3Schema
	structural, err := structuralschema.NewStructural(schema)
	if err != nil {
		t.Fatal(err)
	}
	if errs := structuralschema.ValidateStructural(field.NewPath("schema"), structural); len(errs) != 0 {
		t.Fatalf("nonstructural schema: %v", errs)
	}
	validator, _, err := validation.NewSchemaValidator(schema)
	if err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile("../manifest-examples/minikube.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var example map[string]interface{}
	if err := yaml.Unmarshal(data, &example); err != nil {
		t.Fatal(err)
	}
	if errs := validation.ValidateCustomResource(field.NewPath("example"), example, validator); len(errs) != 0 {
		t.Fatalf("invalid example: %v", errs)
	}
	var rule labelerv1alpha1.Labeler
	if err := yaml.UnmarshalStrict(data, &rule); err != nil {
		t.Fatal(err)
	}
	if !rule.Spec.DryRun || len(rule.Spec.Merge.Taints) != 1 || len(rule.Spec.NodeSelectorTerms) == 0 {
		t.Fatal("example does not decode into the operator API")
	}
	// Keep broader ObjectMeta/NodeSpec fields and unknown merge keys on storage;
	// the v1 schema must not silently prune the original flattened API.
	spec := example["spec"].(map[string]interface{})
	merge := spec["merge"].(map[string]interface{})
	merge["name"] = "node"
	merge["generateName"] = "node-"
	merge["namespace"] = "default"
	merge["selfLink"] = "/api/v1/nodes/node"
	merge["uid"] = "node-uid"
	merge["resourceVersion"] = "7"
	merge["generation"] = int64(2)
	merge["creationTimestamp"] = "2026-10-02T00:00:00Z"
	merge["deletionTimestamp"] = "2026-10-03T00:00:00Z"
	merge["deletionGracePeriodSeconds"] = int64(30)
	merge["podCIDR"] = "10.2.0.0/24"
	merge["podCIDRs"] = []interface{}{"10.2.0.0/24", "fd00::/64"}
	merge["providerID"] = "example://node"
	merge["externalID"] = "legacy-node"
	merge["unschedulable"] = true
	merge["finalizers"] = []interface{}{"example.org/owned"}
	merge["ownerReferences"] = []interface{}{map[string]interface{}{"apiVersion": "v1", "kind": "Node", "name": "owner", "uid": "abc", "controller": true, "blockOwnerDeletion": false, "unknownOwnerField": "keep"}}
	merge["managedFields"] = []interface{}{map[string]interface{}{
		"manager": "example", "operation": "Apply", "apiVersion": "v1", "time": "2026-10-02T00:00:00Z",
		"fieldsType": "FieldsV1", "fieldsV1": map[string]interface{}{"f:spec": map[string]interface{}{"f:unschedulable": map[string]interface{}{}}}, "subresource": "", "unknownManagedField": "keep",
	}}
	merge["configSource"] = map[string]interface{}{"unknownConfigField": "keep", "configMap": map[string]interface{}{
		"namespace": "default", "name": "config", "uid": "config-uid", "resourceVersion": "8", "kubeletConfigKey": "kubelet", "unknownConfigMapField": "keep",
	}}
	merge["podPreemptionPolicy"] = map[string]interface{}{"disableResizePreemption": []interface{}{"example.org/owner"}, "unknownPolicyField": "keep"}
	merge["unknownExtension"] = map[string]interface{}{"nested": []interface{}{"keep"}}
	before, _ := json.Marshal(merge)
	pruning.Prune(example, structural, true)
	after, _ := json.Marshal(spec["merge"])
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("schema pruned merge fields: %s -> %s", before, after)
	}
	if errs := validation.ValidateCustomResource(field.NewPath("example"), example, validator); len(errs) != 0 {
		t.Fatalf("broader API fields rejected: %v", errs)
	}
	data, err = json.Marshal(example)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &rule); err != nil {
		t.Fatalf("valid broader API fields do not decode: %v", err)
	}
	for _, mergeJSON := range []string{
		`{"creationTimestamp":null,"deletionTimestamp":null,"deletionGracePeriodSeconds":null,"ownerReferences":[{"controller":null,"blockOwnerDeletion":null}],"managedFields":[{"time":null,"fieldsV1":null}],"configSource":null,"podPreemptionPolicy":null,"taints":[{"key":"example.org/owned","effect":"NoExecute","timeAdded":null}]}`,
		`{"configSource":{"configMap":null}}`,
	} {
		var nullable map[string]interface{}
		if err := json.Unmarshal([]byte(mergeJSON), &nullable); err != nil {
			t.Fatal(err)
		}
		obj := map[string]interface{}{"spec": map[string]interface{}{"merge": nullable}}
		if errs := validation.ValidateCustomResource(field.NewPath("example"), obj, validator); len(errs) != 0 {
			t.Fatalf("nullable API fields rejected: %v", errs)
		}
		var decoded labelerv1alpha1.MergeSpec
		if err := json.Unmarshal([]byte(mergeJSON), &decoded); err != nil {
			t.Fatalf("nullable API fields do not decode: %v", err)
		}
	}
	// Admission must reject malformed known fields before typed informer decoding.
	for _, tc := range []struct{ name, merge string }{
		{"unschedulable", `{"unschedulable":"true"}`},
		{"podCIDR", `{"podCIDR":true}`},
		{"podCIDRs", `{"podCIDRs":"10.2.0.0/24"}`},
		{"podCIDRs item", `{"podCIDRs":[true]}`},
		{"providerID", `{"providerID":true}`},
		{"externalID", `{"externalID":true}`},
		{"name", `{"name":true}`},
		{"generateName", `{"generateName":true}`},
		{"namespace", `{"namespace":true}`},
		{"selfLink", `{"selfLink":true}`},
		{"uid", `{"uid":true}`},
		{"resourceVersion", `{"resourceVersion":7}`},
		{"generation", `{"generation":"2"}`},
		{"fractional generation", `{"generation":1.5}`},
		{"creationTimestamp", `{"creationTimestamp":true}`},
		{"invalid creationTimestamp", `{"creationTimestamp":"yesterday"}`},
		{"deletionTimestamp", `{"deletionTimestamp":true}`},
		{"invalid deletionTimestamp", `{"deletionTimestamp":"tomorrow"}`},
		{"deletionGracePeriodSeconds", `{"deletionGracePeriodSeconds":"30"}`},
		{"finalizers", `{"finalizers":"example.org/owned"}`},
		{"finalizers item", `{"finalizers":[true]}`},
		{"ownerReferences", `{"ownerReferences":{}}`},
		{"ownerReferences item", `{"ownerReferences":["owner"]}`},
		{"owner apiVersion", `{"ownerReferences":[{"apiVersion":true}]}`},
		{"owner kind", `{"ownerReferences":[{"kind":true}]}`},
		{"owner name", `{"ownerReferences":[{"name":true}]}`},
		{"owner uid", `{"ownerReferences":[{"uid":true}]}`},
		{"owner controller", `{"ownerReferences":[{"controller":"true"}]}`},
		{"owner blockOwnerDeletion", `{"ownerReferences":[{"blockOwnerDeletion":"false"}]}`},
		{"managedFields", `{"managedFields":{}}`},
		{"managedFields item", `{"managedFields":[true]}`},
		{"managed manager", `{"managedFields":[{"manager":true}]}`},
		{"managed operation", `{"managedFields":[{"operation":true}]}`},
		{"managed apiVersion", `{"managedFields":[{"apiVersion":true}]}`},
		{"managed time", `{"managedFields":[{"time":true}]}`},
		{"invalid managed time", `{"managedFields":[{"time":"yesterday"}]}`},
		{"managed fieldsType", `{"managedFields":[{"fieldsType":true}]}`},
		{"managed fieldsV1", `{"managedFields":[{"fieldsV1":"fields"}]}`},
		{"managed subresource", `{"managedFields":[{"subresource":true}]}`},
		{"configSource", `{"configSource":"config"}`},
		{"configMap", `{"configSource":{"configMap":"config"}}`},
		{"config namespace", `{"configSource":{"configMap":{"namespace":true}}}`},
		{"config name", `{"configSource":{"configMap":{"name":true}}}`},
		{"config uid", `{"configSource":{"configMap":{"uid":true}}}`},
		{"config resourceVersion", `{"configSource":{"configMap":{"resourceVersion":7}}}`},
		{"config kubeletConfigKey", `{"configSource":{"configMap":{"kubeletConfigKey":true}}}`},
		{"podPreemptionPolicy", `{"podPreemptionPolicy":"policy"}`},
		{"disableResizePreemption", `{"podPreemptionPolicy":{"disableResizePreemption":true}}`},
		{"disableResizePreemption item", `{"podPreemptionPolicy":{"disableResizePreemption":[true]}}`},
		{"labels", `{"labels":{"wrong":true}}`},
		{"annotations", `{"annotations":{"wrong":true}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var malformed map[string]interface{}
			if err := json.Unmarshal([]byte(tc.merge), &malformed); err != nil {
				t.Fatal(err)
			}
			obj := map[string]interface{}{"spec": map[string]interface{}{"merge": malformed}}
			if errs := validation.ValidateCustomResource(field.NewPath("example"), obj, validator); len(errs) == 0 {
				t.Fatalf("schema accepted malformed merge: %s", tc.merge)
			}
		})
	}
	spec["dryRun"] = "true"
	if errs := validation.ValidateCustomResource(field.NewPath("example"), example, validator); len(errs) == 0 {
		t.Fatal("schema accepted string dryRun")
	}
}
