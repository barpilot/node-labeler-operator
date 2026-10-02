# node-labeler-operator

Restore labels, annotations and taints when Kubernetes replaces your nodes. A
cluster-scoped `Labeler` selects nodes by their existing labels and merges the
configured attributes onto matching nodes.

[![CI](https://github.com/barpilot/node-labeler-operator/actions/workflows/ci.yaml/badge.svg)](https://github.com/barpilot/node-labeler-operator/actions/workflows/ci.yaml)

This is an alpha project. The build targets Kubernetes 1.37 APIs and uses Go
1.26 or newer. CI checks Go 1.26.8 and 1.27.1. Local fake-client tests and schema
validation do not establish compatibility with a live cluster.

## Build and try a rule

```sh
make test
make build
./bin/node-labeler-operator --help
```

The installer creates the CRD; the running operator only reads Labelers and
reads/updates nodes. With an administrative kubeconfig, install the CRD and
start the operator locally:

```sh
kubectl apply -f deploy/crd.yaml
kubectl wait --for=condition=Established --timeout=60s crd/labelers.labeler.barpilot.io
./bin/node-labeler-operator --kubeconfig "$HOME/.kube/config"
```

In another terminal, create the example rule. It selects Linux minikube nodes
(or nodes with `another-node-label-key`) and starts in **dry-run mode**, logging
intended changes without writing nodes:

```sh
kubectl apply -f manifest-examples/minikube.yaml
```

Adjust the selectors and merge values in that file for your nodes. Set
`spec.dryRun: false` and apply the file again when you want enforcement.

## Configuration and ownership

```yaml
apiVersion: labeler.barpilot.io/v1alpha1
kind: Labeler
metadata:
  name: workers
spec:
  dryRun: true
  nodeSelectorTerms:
    - matchExpressions:
        - key: kubernetes.io/os
          operator: In
          values: [linux]
        - key: example.org/pool
          operator: In
          values: [workers]
  merge:
    labels:
      example.org/workload: batch
    annotations:
      example.org/managed-by: node-labeler-operator
    taints:
      - key: example.org/dedicated
        value: batch
        effect: PreferNoSchedule
```

Expressions within a term are ANDed; terms are ORed. Supported label operators
are `In`, `NotIn`, `Exists`, `DoesNotExist`, `Gt` and `Lt`. Empty selectors and
empty terms match no nodes. Invalid expressions do not match. Selection uses
`matchExpressions`; field-based `matchFields` selection is not supported.

| Attribute | Merge behavior |
| --- | --- |
| Labels and annotations | Add missing keys. Existing values win, including empty strings. |
| Taints | Add missing `(key, effect)` pairs. Existing values and system taints are preserved. Repeated syncs do not duplicate taints. |
| Other metadata | Fill unset fields; append missing metadata list entries without duplicating identical entries. Existing scalar values win. |
| Other node spec fields | Preserve the original nonzero overwrite behavior. For example, `podCIDR`, `podCIDRs` and `providerID` can replace existing values, and `unschedulable: true` can cordon a node. Empty strings, empty lists and `false` do not clear existing values. Non-null spec pointers replace the corresponding values. |

`merge` retains the original **flattened** `ObjectMeta` and `NodeSpec` API:
fields such as `finalizers`, `ownerReferences`, `podCIDR` and `unschedulable` sit
beside `labels` and `taints`, without nested `metadata` or `spec` objects. These
broader fields still pass through the merge and normal Kubernetes admission
and validation; immutable or obsolete fields can be rejected by the API
server. The operator does not manage cleanup for finalizers supplied by a rule.
The CRD preserves unknown merge fields on storage to avoid pruning existing
configurations. Only fields represented by the current Kubernetes `ObjectMeta`
and `NodeSpec` types are acted on; `status` is not supported and no node status
subresource is written.

For overlapping rules, an existing label/annotation or taint value wins. If two
rules request different values for the same missing key (or taint key/effect),
**the first successful update wins**. Rule processing order is not guaranteed;
reversing the order can produce a different result. Avoid conflicting rules.
Other spec fields that overwrite can also conflict between rules.

Editing or deleting a Labeler stops its previous controller. Attributes already
applied remain on nodes, including when a node stops matching a selector.
`dryRun: true` stops writes for that rule; it does not undo earlier changes.
Updates already accepted by the API server may complete during shutdown.

## Run in a cluster

Build an image from this checkout. Published historical images do not contain
these changes:

```sh
docker build -t node-labeler-operator:local .
minikube image load node-labeler-operator:local
kubectl apply -f deploy/crd.yaml
kubectl wait --for=condition=Established --timeout=60s crd/labelers.labeler.barpilot.io
kubectl apply -f deploy/rbac.yaml -f deploy/deployment.yaml
```

For another cluster, publish the image to your registry and change the image
in `deploy/deployment.yaml` before applying it. The container runs as a nonroot
user with a read-only filesystem. The supplied deployment uses one replica
and `Recreate` updates: there is no leader election. Each Labeler has its own
node watch, so memory and API traffic grow with the number of rules and nodes.

Only trusted administrators should be able to create or edit Labelers: rules
can change cluster-wide node metadata and scheduling. Runtime RBAC grants
`get/list/watch/update` on nodes and `get/list/watch` on Labelers; it grants no
CRD administration or `nodes/status` access.

To stop enforcement, delete the deployment. To remove individual rules, delete
the Labeler objects. Neither operation removes previously applied attributes.
Deleting the CRD also deletes its Labeler objects; export rules first if they
are needed later.

## Development

Go modules and checksums replace the old `dep` vendor tree. Generated clients
and deep-copy methods are checked in; their generators are pinned in `go.mod`:

```sh
go mod download
make generate
make test
go vet ./...
make build
go run github.com/yannh/kubeconform/cmd/kubeconform@v0.8.0 -strict -kubernetes-version 1.37.0 -summary deploy/rbac.yaml deploy/deployment.yaml
```

Tests cover merge ownership and idempotence, dry-run, conflict retries,
controller shutdown on edit/delete, and CRD validation/pruning using Kubernetes
API-server libraries. CI also checks generated-file drift, manifests, and a
container build/help smoke test. No cluster credentials are needed for these
checks. `--resync-seconds` (default 30) controls rule resync; node controllers
resync every 30 seconds and process watch events between resyncs.

When upgrading an existing installation, apply the v1 CRD explicitly before
starting this version. The CRD group, kind, scope and `v1alpha1` resource version
are unchanged. Empty-string labels and annotations now count as existing values
and remain unchanged; the old mergo behavior filled them from the rule.
Taints now use additive merging instead of replacing the entire list; existing
taints are retained. `dryRun` is now enforced, and rule
edits/deletion actually stop the previous node watch.
