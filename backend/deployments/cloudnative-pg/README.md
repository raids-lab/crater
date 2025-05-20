
## Install the Operator 

```shell
helm repo add cnpg https://cloudnative-pg.github.io/charts

helm upgrade --install cnpg \
  --namespace crater \
  --create-namespace \
  --set config.clusterWide=false \
  cnpg/cloudnative-pg \
  -f values.yaml
```

## Install  DB Cluster

We download the charts and modify `ping.yaml`, so may use the charts directly（v0.1.0）: 

# Fetch the cluster chart and unpack it to ./cluster
helm pull cnpg/cluster --untar --untardir ./cluster

modify `./cluster/templates/tests/ping.yaml`

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "cluster.fullname" . }}-ping-test
  labels:
    app.kubernetes.io/component: database-ping-test
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
spec:
  template:
    metadata:
      name: {{ include "cluster.fullname" . }}-ping-test
      labels:
        app.kubernetes.io/component: database-ping-test
    spec:
      restartPolicy: Never
      containers:
        - name: alpine
          image: {{ .Values.cluster.pingTestImageName }}
          command: [ 'sh' ]
          env:
            - name: PGUSER
              valueFrom:
                secretKeyRef:
                  name: {{ include "cluster.fullname" . }}-app
                  key: username
            - name: PGPASS
              valueFrom:
                secretKeyRef:
                  name: {{ include "cluster.fullname" . }}-app
                  key: password
          args:
            - "-c"
            - >-
              apk add postgresql-client &&
              psql "postgresql://$PGUSER:$PGPASS@{{ include "cluster.fullname" . }}-rw.{{ .Release.Namespace }}.svc.cluster.local:5432" -c 'SELECT 1'

```

```shell
helm show values cnpg/cluster

helm upgrade --install database \
  --namespace crater \
  ./cluster \
  -f cluster.values.yaml --dry-run
```

## Fetch DB password

```shell
kubectl get secret database-cluster-app -n crater -o jsonpath="{.data.password}" | base64 --decode
```

Keep the password for crater deployments!!

## TroubleShooting

### openebs pvc

PVC provision in WaitForFirstConsumer mode may cause Deadlock on pod scheduling!

https://github.com/openebs/openebs/issues/2915

we add custom labels to nodes in cluster for dependecy components running, and the affinity is configured in values:

```shell
kubectl label node your-devops-node crater.raids-lab.io/devops="true"
```