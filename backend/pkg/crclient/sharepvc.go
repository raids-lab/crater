package crclient

import (
	"context"
	"fmt"

	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ShareDir struct {
	Pvc       string
	Namespace string
	pv        *corev1.PersistentVolume
}

type PVCClient struct {
	client.Client
}

var shareDirs map[string]ShareDir = make(map[string]ShareDir)

// var shareDirs map[string]ShareDir = map[string]ShareDir{
// 	"dnn-train-data": {
// 		Pvc:       "dnn-train-data",
// 		Namespace: "user-wjh",
// 	},
// 	"jupyterhub-shared-volume": {
// 		Pvc:       "jupyterhub-shared-volume",
// 		Namespace: "jupyter",
// 	},
// }

func (c *PVCClient) InitShareDir() error {
	// c.Client.
	shareDirList := config.GetShareDirs()
	for _, shareDir := range shareDirList {
		pv, err := c.GetPVCRelatedPV(shareDir.Pvc, shareDir.Namespace)
		if err != nil {
			logrus.Errorf("get share dir pv failed: %v", err)
			return err
		}
		shareDirs[shareDir.Pvc] = ShareDir{
			Pvc:       shareDir.Pvc,
			Namespace: shareDir.Namespace,
			pv:        pv,
		}
	}
	logrus.Infof("init share dirs success: %v", shareDirs)
	return nil
}

func (c *PVCClient) CheckOrCreateUserPvc(userNamespace, pvcName string) error {
	_, err := c.GetPVC(pvcName, userNamespace)
	if err == nil {
		return nil
	}
	return c.createUserPVCFromShareDir(userNamespace, pvcName)
}

func (c *PVCClient) createUserPVCFromShareDir(userNamespace, pvcName string) error {
	if _, ok := shareDirs[pvcName]; !ok {
		logrus.Errorf("share dir not found: %s", pvcName)
		return fmt.Errorf("share dir not found")
	}
	sharepv := shareDirs[pvcName].pv

	volumeAttr := sharepv.Spec.CSI.VolumeAttributes
	volumeAttr["staticVolume"] = "true"
	volumeAttr["rootPath"] = volumeAttr["subvolumePath"]
	volumeName := fmt.Sprintf("%s-%s", sharepv.Name, userNamespace)
	newpv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeName,
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes:                   sharepv.Spec.AccessModes,
			Capacity:                      sharepv.Spec.Capacity,
			VolumeMode:                    sharepv.Spec.VolumeMode,
			StorageClassName:              sharepv.Spec.StorageClassName,
			PersistentVolumeReclaimPolicy: sharepv.Spec.PersistentVolumeReclaimPolicy,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver:           sharepv.Spec.CSI.Driver,
					VolumeHandle:     fmt.Sprintf("%s-%s", sharepv.Spec.CSI.VolumeHandle, userNamespace),
					VolumeAttributes: volumeAttr,
					NodeStageSecretRef: &corev1.SecretReference{
						Name:      "rook-csi-cephfs-node-user",
						Namespace: "rook-ceph",
					},
				},
			},
		},
	}
	newpvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: userNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			VolumeName:       volumeName,
			VolumeMode:       sharepv.Spec.VolumeMode,
			StorageClassName: &sharepv.Spec.StorageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: sharepv.Spec.Capacity,
			},
		},
	}
	if err := c.Client.Create(context.Background(), newpv); err != nil {
		return err
	}
	if err := c.Client.Create(context.Background(), newpvc); err != nil {
		return err
	}
	logrus.Infof("create user pvc from share dir success, pvc:%v, namespace:%v", pvcName, userNamespace)
	return nil
}

func (c *PVCClient) GetPVC(name, namespace string) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, pvc); err != nil {
		return nil, err
	}
	return pvc, nil
}

func (c *PVCClient) GetPVCRelatedPV(name, namespace string) (*corev1.PersistentVolume, error) {
	pvc := &corev1.PersistentVolumeClaim{}

	if err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, pvc); err != nil {
		return nil, err
	}
	if pvc.Spec.VolumeName == "" {
		return nil, fmt.Errorf("volume name empty")
	}
	pv := &corev1.PersistentVolume{}
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "", Name: pvc.Spec.VolumeName}, pv); err != nil {
		return nil, err
	}
	if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		if err := c.Update(context.Background(), pv); err != nil {
			logrus.Errorf("update pv %s reclaimPolicy  failed: %v", name, err)
			return nil, err
		}
	}

	return pv, nil
}
