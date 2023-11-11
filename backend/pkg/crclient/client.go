package crclient

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// errors
	"k8s.io/apimachinery/pkg/api/errors"
)

type Control struct {
	client.Client
}

// todo: add more volumes, args etc..
func (c *Control) CreateUserNameSpace(ns string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	err := c.Create(context.Background(), namespace)
	// already exists
	if errors.IsAlreadyExists(err) {
		logrus.Infof("namespace %s already exists", ns)
		return nil
	}
	if err != nil {
		return fmt.Errorf("create namespace %s failed: %v", ns, err)
	}
	return nil
}
func (c *Control) CreateUserHomePVC(namespace string, pvcname string) error {
	SCN := "rook-cephfs"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcname,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": resource.MustParse("50Gi"),
				},
			},
			StorageClassName: &SCN,
		},
	}

	err := c.Create(context.Background(), pvc)
	if errors.IsAlreadyExists(err) {
		logrus.Infof("pvc %s already exists", pvcname)
		return nil
	}
	if err != nil {
		return fmt.Errorf("create pvc %s failed: %v", pvcname, err)
	}
	return nil
}
