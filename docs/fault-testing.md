# Fault testing

## StatefulSet pod recovery

### Setup

- Kubernetes context: `docker-desktop`
- Kubernetes node: `desktop-control-plane` (`Ready`)
- Raw Raft StatefulSet: three replicas
- Before the test, all three Raft pods were `1/1 Running`.

### Test

The latest observed Raft election log showed node 1 as leader (term 39). Delete that leader pod:

```powershell
kubectl delete pod raft-node-1
kubectl wait --for=condition=Ready pod/raft-node-1 --timeout=180s
kubectl get pods -o wide
```

### Observed result

- Kubernetes deleted `raft-node-1`.
- The StatefulSet created a replacement `raft-node-1`.
- The replacement became `1/1 Running`; the StatefulSet returned to `3/3 Ready`.
- The replacement received a new pod IP.
- Node 2 logged successful heartbeat/replication calls to peers after the replacement was ready, including peer 1.

### Conclusion and limits

Kubernetes pod recreation and Raft peer reconnection were observed. This run did not verify client-visible parking availability because the parking gRPC service is not deployed in the Raft StatefulSet and is not integrated with Raft. Raft state is in memory, so this is not evidence of durable recovery or production fault tolerance.

## Repeatable command

After verifying a healthy three-node deployment, the same recreation check can be repeated with:

```powershell
kubectl delete pod raft-node-1
kubectl wait --for=condition=Ready pod/raft-node-1 --timeout=180s
kubectl get pods -o wide
kubectl logs raft-node-1 --tail=50
```
