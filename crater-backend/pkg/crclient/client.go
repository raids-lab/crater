package crclient

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Control struct {
	client.Client
}

// todo: add more volumes, args etc..
func (c *Control) CreateNameSpace(ns string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	err := c.Create(context.Background(), namespace)
	if err != nil {
		return fmt.Errorf("create namespace %s failed: %v", ns, err)
	}
	return nil
}
func (c *Control) CreatePVC(namespace string, pvcname string) error {
	pvcSrc := new(corev1.PersistentVolumeClaim)
	pvcSrc.ObjectMeta.Name = pvcname
	pvcSrc.Spec.AccessModes = append(pvcSrc.Spec.AccessModes, corev1.ReadWriteMany)

	//设置存储大小
	var resourceQuantity resource.Quantity
	resourceQuantity.Set(50 * 1024 * 1024 * 1024)
	pvcSrc.Spec.Resources.Requests = corev1.ResourceList{
		"storage": resourceQuantity,
	}
	var SCN = "rook-cephfs"
	//使用存储卷名字
	//if len(request.StorageClassName) != 0 {
	pvcSrc.Spec.StorageClassName = &SCN
	//}

	err := c.Create(context.Background(), pvcSrc)
	if err != nil {
		return fmt.Errorf("create pvc %s failed: %v", pvcname, err)
	}
	return nil
}
