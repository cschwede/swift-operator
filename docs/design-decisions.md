# Design decisions

## Storage StatefulSet

Swift needs storage devices for the actual data, and these need to be
accessible using the same hostname (or IP) during their lifetime. A StatefulSet
in combination with a Headless Service is exactly what we need in this case.

From
[https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/):

> If you want to use storage volumes to provide persistence for your workload,
> you can use a StatefulSet as part of the solution. Although individual Pods
> in a StatefulSet are susceptible to failure, the persistent Pod identifiers
> make it easier to match existing volumes to the new Pods that replace any
> that have failed.

Swift requires quite a few services to access these PVs, and all of them are
running in a single pod.

Additionally, volumes are not deleted if the StatefulSet is deleted. This is a
very welcome safety limitation - an unwanted removal of the StatefulSet (or the
whole deployment) will not immediately result in a catastrophic data loss, but
can be recovered from (though this might need human interaction to fix).

### Headless service

The headless service makes it possible to access the storage pod (and
especially the `*-server` processes) directly by using a DNS name. For example,
if the pod name is `swift-storage-0` and the SwiftStorage instance is named
`swift-storage`, it becomes accessible using `swift-storage-0.swift-storage`.
This makes it easily usable within the Swift rings, and IP changes are now
transparent and don't require an update of the rings.

### PodManagementPolicy
From
[https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#parallel-pod-management](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#parallel-pod-management)

>Parallel pod management tells the StatefulSet controller to launch or
>terminate all Pods in parallel, and to not wait for Pods to become Running and
>Ready or completely terminated prior to launching or terminating another Pod.
>This option only affects the behavior for scaling operations. Updates are not
>affected.

This is required to scale by more than one (including new deployments with more
than one replica). It is required to create all pods at the same time,
otherwise there will be PVCs that are not bound and the Swift rings can't be
created, eventually blocking the start of these pods.

### QoS class and resource limits

By default, SwiftStorage and SwiftProxy pods use the "BestEffort" QoS class.
However, these pods might be evicted if the system is under pressure. To prevent
this, only the API server containers (`*-server`) have resource limits set,
which promotes the pod to the "Burstable" QoS class. This prevents eviction
while an API request is ongoing. The `*-server` processes also use significantly
more memory compared to the background processes (replicators, auditors, etc.).

Using the "Guaranteed" QoS class would require setting resource limits on all
containers. Due to the large number of containers in each pod, this would either
require overly fine-grained limits or allocating far more resources than actually
needed for the background processes. Kubernetes 1.34+ will support pod-level QoS
settings, which would allow setting a single resource limit for the entire pod
instead of per container, making the "Guaranteed" QoS class practical.

### Scale-in prevention

The webhook rejects updates that reduce the number of SwiftStorage replicas.
Scaling down would leave orphaned PVCs and risk data loss, because all data from
the removed disks needs to be redistributed to the remaining ones. This could
overfill the remaining PVs if there is not enough free capacity. Scale-up is
allowed, making scaling intentionally asymmetric.

## Affinity
Storage pods should be distributed to different nodes to avoid single points of
failure. A podAntiAffinity rule with
preferredDuringSchedulingIgnoredDuringExecution is used to distribute pods to
different nodes if possible. Using a separate storageClass and
PersistentVolumes that are located on different nodes can be used to enforce
further distribution.

## NetworkPolicy & Labels

Swift backend services must only be accessible by other backend services and
the Swift proxy. To limit access, a NetworkPolicy is added to allow only
traffic between these pods. The NetworkPolicy itself depends on labels, and
these must match to allow traffic. Therefore labels must not be unique; instead
all pods must use the same label to allow access. This is also the reason why
the swift-operator is not using labels from
[lib-common](https://github.com/openstack-k8s-operators/lib-common) (yet).

## Swift hash path prefix and suffix

Swift uses `swift_hash_path_prefix` and `swift_hash_path_suffix` values in
`swift.conf` to determine the placement of data on disks. These are generated
once using random strings when the Secret is first created and must never change
— if they do, all existing data becomes inaccessible. The controller checks if
the Secret already exists before creating it, which also allows operators to
pre-create the Secret with specific values when adopting an existing deployment.

## Swift rings

### Ring ConfigMap ownership

The ring file ConfigMap is created by the ring rebalance job (swift-ring-tool),
not by the controller. Therefore the owner reference is set by the job itself,
which POSTs the ConfigMap with `ownerReferences` and a finalizer via the
Kubernetes API. The controller only removes the finalizer during SwiftRing
deletion.

Swift rings require information about the disks to use, and this includes
sizes (weights) and hostnames (or IPs). Sizes are not known when starting the
StatefulSet using PVCs - the size requirement is a lower limit, but the actual
PVs might be way bigger.

However, StatefulSets do create PVCs before the ConfigMaps are available and
simply wait starting the pods until these become available. The SwiftRing
reconciler is watching the SwiftStorage instances and iterates over PVCs
to get actual information about the used disks. Once these are bound the size
is known, and the swift-ring-rebalance job creates the Swift rings and
eventually the ConfigMap. After the ConfigMap becomes available, StatefulSets
will start the service pods.

### Ring synchronization

Rings are stored in a ConfigMap mounted by the SwiftProxy and SwiftStorage
instances using projected volumes. This makes it possible to mount all required
files (rings, storage/proxy config files as well as some global files like
swift.conf) at the same place, without merging these from other places. Updated
ConfigMaps will update these files, and these changes are are detected by the
Swift services eventually reloading these.

## Customizing configurations
Some operators are using the `customServiceConfig` option to customize
settings. However, the SwiftRing instance deploys multiple backend services,
and each of these requires specific files to be customized. Therefore only
`defaultConfigOverwrite` using specific keys as filenames is supported when
using the swift-operator.
