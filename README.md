# node-labeler-operator
kubernetes operator to auto label/taint/annotate node based on a CRD.

[![GitHub release](https://img.shields.io/github/release/barpilot/node-labeler-operator.svg)](https://github.com/barpilot/node-labeler-operator/releases)
[![Travis](https://img.shields.io/travis/barpilot/node-labeler-operator.svg)](https://travis-ci.org/barpilot/node-labeler-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/barpilot/node-labeler-operator)](https://goreportcard.com/report/github.com/barpilot/node-labeler-operator)
[![Docker Build Status](https://img.shields.io/docker/build/barpilot/node-labeler-operator.svg)](https://hub.docker.com/r/barpilot/node-labeler-operator/)

NOTE: This is an alpha-status project. We do regular tests on the code and functionality, but we can not assure a production-ready stability.

## Requirements

_node-labeler-operator_ is meant to be run on Kubernetes 1.8+. All dependecies have been vendored, so there's no need to any additional download.

## Usage

### Installation

In order to create _node-labeler-operator_ inside a Kubernetes cluster, the operator has to be deployed. It can be done with a deployment.
```
kubectl run node-labeler-operator --image=barpilot/node-labeler-operator --namespace=kube-system
```

### Configuration

_node-labeler-operator_ is using a [CRD](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) for its configuration.
Here is a description of an object:
```yaml
apiVersion: labeler.barpilot.io/v1alpha1
kind: Labeler
metadata:
  name: example
  labels:
    operator: node-labeler-operator
spec:
  nodeSelectorTerms:
  - matchExpressions:
    - key: kubernetes.io/hostname
      operator: In
      values:
      - minikube
    - key: beta.kubernetes.io/os
      operator: In
      values:
      - linux
  - matchExpressions:
    - key: another-node-label-key
      operator: Exists
  merge:
    metadata:
      labels:
        mylabel: "true"
      annotations:
        node-labeler-operator: works
    spec:
      taints:
        - key: dedicated
          value: foo
          effect: PreferNoSchedule
    status:
      capacity:
        example.com/dongle: "4"
        example.com/token: "1"
```
for more information about `nodeSelectorTerms` have a look at: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/

### Cases

- VM on private cloud provider.  
Nodes are removed on shutdown and so lose theirs attributes.

## Features
- [x] Node selection
- [x] Adding attributes
  - [x] Labels
  - [x] Annotations
  - [x] Taints
  - [x] Status
- [ ] Removing attributes
- [ ] Overwrite attributes
