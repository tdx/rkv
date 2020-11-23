#!/bin/sh
kind create cluster --config cluster.yaml

# make worker roles
kubectl label node kind-worker  node-role.kubernetes.io/worker=
kubectl label node kind-worker2 node-role.kubernetes.io/worker=
kubectl label node kind-worker3 node-role.kubernetes.io/worker=

# load balancer
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.9.5/manifests/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.9.5/manifests/metallb.yaml
## On first install only
kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)"

# find cluster network
#kubectl get nodes -o wide
#NAME                 STATUS   ROLES    AGE   VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE       KERNEL-VERSION      CONTAINER-RUNTIME
#kind-control-plane   Ready    master   20m   v1.18.2   172.18.0.2    <none>        Ubuntu 19.10   4.4.0-142-generic   containerd://1.3.3-14-g449e9269
#kind-worker          Ready    <none>   19m   v1.18.2   172.18.0.3    <none>        Ubuntu 19.10   4.4.0-142-generic   containerd://1.3.3-14-g449e9269
#
# cluster network is 172.18.0.0/16
#  check 'addresses' in load-balancer.yaml

kubectl apply -f load-balancer.yaml
#helm install rkvd ../deploy/rkvd
