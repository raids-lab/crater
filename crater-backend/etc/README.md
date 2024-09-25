# How to update release config

```bash
kubectl delete configmap backend-config -n crater
kubectl create configmap backend-config -n crater --from-file=config.yaml=etc/zjlab-config.yaml 
```