# ai-task-controller

## crd 创建

```bash
kubectl apply -f config/crd/bases/aisystem.github.com_aijobs.yaml 
```


## 数据库部署和连接

* `deploy/mysql`： 在集群上部署mysql集群
* `dbconf.yaml`： 数据库配置文件，可以通过 `kubectl port-forward service/mycluster mysql` 在本机上暴露3306端口


## 运行

``` bash
# 编译
go mod tidy && go build -o bin/controller main.go

# 本地运行
./bin/controller --db-config-file ./debug-dbconf.yaml
```

## 开发与部署规范

### 开发与自测

1. 从main中checkout出分支单独开发
2. 完成开发后可以先用postman自测
	- 在***REMOVED***，使用`debug.sh`启动后端，默认端口8099，如遇端口冲突请自行修改端口
	- 可以在header中添加`X-Debug-Username`指定用户名绕过登录认证，直接测试接口
3. 自测完成后，若需要与前端联调，则告知前端端口并进行联调
4. 测试完成后合入主分支，gitlab上提mr，发群里，有群友ap后合入即可

### 联调

1. 合入main分支后，在***REMOVED***运行`docker restart ai-portal-backend`，会启动main分支版本，暴露8078端口
2. 通知前端联调前端功能

### 部署

- 建议上线前群里沟通好
- 打包：`make build-backend IMG={YOUR IMAGE PATH}`
- 部署：`make deploy-backend IMG={YOUR IMAGE PATH}`
- 验证：联系前端部署到k8s里，前端部署完成后进行整体的功能验证

This repository implements a simple controller for watching Foo resources as
defined with a CustomResourceDefinition (CRD).

**Note:** go-get or vendor this package as `k8s.io/sample-controller`.

This particular example demonstrates how to perform basic operations such as:

* How to register a new custom resource (custom resource type) of type `Foo` using a CustomResourceDefinition.
* How to create/get/list instances of your new resource type `Foo`.
* How to setup a controller on resource handling create/update/delete events.

It makes use of the generators in [k8s.io/code-generator](https://github.com/kubernetes/code-generator)
to generate a typed client, informers, listers and deep-copy functions. You can
do this yourself using the `./hack/update-codegen.sh` script.

The `update-codegen` script will automatically generate the following files &
directories:

* `pkg/apis/samplecontroller/v1alpha1/zz_generated.deepcopy.go`
* `pkg/generated/`

Changes should not be made to these files manually, and when creating your own
controller based off of this implementation you should not copy these files and
instead run the `update-codegen` script to generate your own.

## Details

The sample controller uses [client-go library](https://github.com/kubernetes/client-go/tree/master/tools/cache) extensively.
The details of interaction points of the sample controller with various mechanisms from this library are
explained [here](docs/controller-client-go.md).

## Fetch sample-controller and its dependencies

Like the rest of Kubernetes, sample-controller has used
[godep](https://github.com/tools/godep) and `$GOPATH` for years and is
now adopting go 1.11 modules.  There are thus two alternative ways to
go about fetching this demo and its dependencies.

### Fetch with godep

When NOT using go 1.11 modules, you can use the following commands.

```sh
go get -d k8s.io/sample-controller
cd $GOPATH/src/k8s.io/sample-controller
godep restore
```

### When using go 1.11 modules

When using go 1.11 modules (`GO111MODULE=on`), issue the following
commands --- starting in whatever working directory you like.

```sh
git clone https://github.com/kubernetes/sample-controller.git
cd sample-controller
```

Note, however, that if you intend to
generate code then you will also need the
code-generator repo to exist in an old-style location.  One easy way
to do this is to use the command `go mod vendor` to create and
populate the `vendor` directory.

### A Note on kubernetes/kubernetes

If you are developing Kubernetes according to
https://github.com/kubernetes/community/blob/master/contributors/guide/github-workflow.md
then you already have a copy of this demo in
`kubernetes/staging/src/k8s.io/sample-controller` and its dependencies
--- including the code generator --- are in usable locations
(valid for all Go versions).

## Purpose

This is an example of how to build a kube-like controller with a single type.

## Running

**Prerequisite**: Since the sample-controller uses `apps/v1` deployments, the Kubernetes cluster version should be greater than 1.9.

```sh
# assumes you have a working kubeconfig, not required if operating in-cluster
go build -o sample-controller .
./sample-controller -kubeconfig=$HOME/.kube/config

# create a CustomResourceDefinition
kubectl create -f artifacts/examples/crd-status-subresource.yaml

# create a custom resource of type Foo
kubectl create -f artifacts/examples/example-foo.yaml

# check deployments created through the custom resource
kubectl get deployments
```

## Use Cases

CustomResourceDefinitions can be used to implement custom resource types for your Kubernetes cluster.
These act like most other Resources in Kubernetes, and may be `kubectl apply`'d, etc.

Some example use cases:

* Provisioning/Management of external datastores/databases (eg. CloudSQL/RDS instances)
* Higher level abstractions around Kubernetes primitives (eg. a single Resource to define an etcd cluster, backed by a Service and a ReplicationController)

## Defining types

Each instance of your custom resource has an attached Spec, which should be defined via a `struct{}` to provide data format validation.
In practice, this Spec is arbitrary key-value data that specifies the configuration/behavior of your Resource.

For example, if you were implementing a custom resource for a Database, you might provide a DatabaseSpec like the following:

``` go
type DatabaseSpec struct {
	Databases []string `json:"databases"`
	Users     []User   `json:"users"`
	Version   string   `json:"version"`
}

type User struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
```

## Validation

To validate custom resources, use the [`CustomResourceValidation`](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/#validation) feature. Validation in the form of a [structured schema](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema) is mandatory to be provided for `apiextensions.k8s.io/v1`.

### Example

The schema in [`crd.yaml`](./artifacts/examples/crd.yaml) applies the following validation on the custom resource:
`spec.replicas` must be an integer and must have a minimum value of 1 and a maximum value of 10.

## Subresources

Custom Resources support `/status` and `/scale` [subresources](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#subresources). The `CustomResourceSubresources` feature is in GA from v1.16.

### Example

The CRD in [`crd-status-subresource.yaml`](./artifacts/examples/crd-status-subresource.yaml) enables the `/status` subresource for custom resources.
This means that [`UpdateStatus`](./controller.go) can be used by the controller to update only the status part of the custom resource.

To understand why only the status part of the custom resource should be updated, please refer to the [Kubernetes API conventions](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).

In the above steps, use `crd-status-subresource.yaml` to create the CRD:

```sh
# create a CustomResourceDefinition supporting the status subresource
kubectl create -f artifacts/examples/crd-status-subresource.yaml
```

## A Note on the API version
The [group](https://kubernetes.io/docs/reference/using-api/#api-groups) version of the custom resource in `crd.yaml` is `v1alpha`, this can be evolved to a stable API version, `v1`, using [CRD Versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/).

## Config

```bash
go run main.go --kubeconfig ~/.kube/config
```