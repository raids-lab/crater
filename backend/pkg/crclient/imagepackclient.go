package crclient

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
)

type ImagePackController struct {
	client.Client
}

func (c *ImagePackController) CreateImagePack(ctx context.Context, imagepack *imagepackv1.ImagePack) error {
	err := c.Create(ctx, imagepack)
	if err != nil {
		return fmt.Errorf("create imagepack: %w", err)
	}
	return nil
}

func (c *ImagePackController) GetImagePack(ctx context.Context, name, namespace string) (*imagepackv1.ImagePack, error) {
	imagepack := &imagepackv1.ImagePack{}
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, imagepack); err != nil {
		return nil, err
	}
	return imagepack, nil
}

func (c *ImagePackController) DeleteImagePackByName(ctx context.Context, name, namespace string) error {
	imagepack, _ := c.GetImagePack(ctx, name, namespace)
	err := c.Delete(ctx, imagepack)
	return err
}

func (c *ImagePackController) DeleteImagePackByEntity(ctx context.Context, imagepack *imagepackv1.ImagePack) error {
	err := c.Delete(ctx, imagepack)
	return err
}

func (c *ImagePackController) ListImagePack(ctx context.Context, namespace string) ([]*imagepackv1.ImagePack, error) {
	imagePackList := &imagepackv1.ImagePackList{}
	if err := c.List(ctx, imagePackList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}
	ret := make([]*imagepackv1.ImagePack, 0, len(imagePackList.Items))
	for i := range imagePackList.Items {
		ret = append(ret, &imagePackList.Items[i])
	}
	return ret, nil
}

func (c *ImagePackController) GetImagePackPod(ctx context.Context, name, namespace string) (*corev1.Pod, error) {
	var podList corev1.PodList
	err := c.List(ctx, &podList, client.MatchingLabels{"job-name": name}, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	} else if len(podList.Items) > 0 {
		pod := podList.Items[0]
		return &pod, nil
	} else {
		return nil, nil
	}
}
