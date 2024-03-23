package crclient

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// errors
	"k8s.io/apimachinery/pkg/api/errors"
)

type Control struct {
	client.Client
}

// todo: add more volumes, args etc..
func (c *Control) CreateUserNameSpace(username string) error {
	ns := NameSpace
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
		return fmt.Errorf("create namespace %s failed: %w", ns, err)
	}
	return nil
}
