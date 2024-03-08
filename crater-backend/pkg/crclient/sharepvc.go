package crclient

import (
	"context"
	"fmt"

	"github.com/aisystem/ai-protal/pkg/config"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PVCClient struct {
	client.Client
}

const (
	NameSpace   = "crater-jobs"
	UserHomePVC = "home-%s-pvc"
	DataPVCName = "data-pvc"
)

type ShareDir struct {
	Pvc       string
	Namespace string
	pv        *corev1.PersistentVolume
}

var shareDirs map[string]ShareDir = make(map[string]ShareDir)

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
		// pvc already exists
		return nil
	}
	if _, ok := shareDirs[pvcName]; !ok {
		logrus.Errorf("share dir not found: %s", pvcName)
		return fmt.Errorf("share dir not found")
	}
	return c.createUserPVCFromPV(shareDirs[pvcName].pv, userNamespace, pvcName)
}

// MigratePvcFromOldNamespace migrates a PVC (Persistent Volume Claim) from an old namespace to a new namespace.
// It checks if the PVC already exists in the new namespace, and if it does, it returns nil.
// If the PVC does not exist in the new namespace, it retrieves the related PV (Persistent Volume) from the new namespace and creates a new PVC in the new namespace using the PV.
// Parameters:
//   - oldNamespace: The old namespace from which the PVC is being migrated.
//   - newNamespace: The new namespace to which the PVC is being migrated.
//   - pvcName: The name of the PVC being migrated.
//
// Returns:
//   - error: An error if any occurred during the migration process, or nil if the migration was successful.
func (c *PVCClient) MigratePvcFromOldNamespace(oldNamespace, newNamespace, oldPvcName, newPvcName string) error {
	_, err := c.GetPVC(newPvcName, newNamespace)
	if err == nil {
		// pvc already exists
		return nil
	}

	pv, err := c.GetPVCRelatedPV(oldPvcName, oldNamespace)
	if err != nil {
		logrus.Errorf("get share dir pv failed: %v", err)
		return err
	}

	return c.createUserPVCFromPV(pv, newNamespace, newPvcName)
}

func (c *PVCClient) createUserPVCFromPV(sharepv *corev1.PersistentVolume, newNamespace, newPvcName string) error {
	volumeAttr := sharepv.Spec.CSI.VolumeAttributes
	volumeAttr["staticVolume"] = "true"
	volumeAttr["rootPath"] = volumeAttr["subvolumePath"]
	volumeName := fmt.Sprintf("%s-%s", sharepv.Name, newNamespace)
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
					VolumeHandle:     fmt.Sprintf("%s-%s", sharepv.Spec.CSI.VolumeHandle, newNamespace),
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
			Name:      newPvcName,
			Namespace: newNamespace,
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
	logrus.Infof("create user pvc from share dir success, pvc:%v, namespace:%v", newPvcName, newNamespace)
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

func (c *PVCClient) CreateUserHomePVC(username string) error {
	namespace := NameSpace
	pvcname := fmt.Sprintf(UserHomePVC, username)

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
