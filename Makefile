
CODE_GENERATOR_IMAGE := quay.io/slok/kube-code-generator:v1.10.0
DIRECTORY := $(PWD)
CODE_GENERATOR_PACKAGE := github.com/barpilot/node-labeler-operator

generate:
	docker run --rm -it \
	-v $(DIRECTORY):/go/src/$(CODE_GENERATOR_PACKAGE) \
	-e PROJECT_PACKAGE=$(CODE_GENERATOR_PACKAGE) \
	-e CLIENT_GENERATOR_OUT=$(CODE_GENERATOR_PACKAGE)/client/k8s \
	-e APIS_ROOT=$(CODE_GENERATOR_PACKAGE)/apis \
	-e GROUPS_VERSION="labeler:v1alpha1" \
	-e GENERATION_TARGETS="deepcopy,client" \
	$(CODE_GENERATOR_IMAGE)
