package crclient

import (
	"bytes"
	"context"

	aijobapi "github.com/aisystem/ai-protal/pkg/apis/aijob/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LogClient struct {
	client.Client
	KubeClient kubernetes.Interface
}

// GetPodsWithLabel 获取具有特定标签的 Pod 列表
func (lc *LogClient) GetPodsWithLabel(namespace string, jobName string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}

	if err := lc.List(context.TODO(), podList, client.InNamespace(namespace), client.MatchingLabelsSelector{
		Selector: labels.SelectorFromSet(map[string]string{aijobapi.JobNameLabel: jobName}),
	}); err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// GetPodLogs 获取指定 Pod 的日志
func (lc *LogClient) GetPodLogs(pod corev1.Pod) (string, error) {
	logOpts := &corev1.PodLogOptions{}
	req := lc.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, logOpts)

	logs, err := req.Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer logs.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(logs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GetSvcPort 获取指定 Service 的 NodePort
func (lc *LogClient) GetSvcPort(namespace string, svcName string) (int32, error) {
	svc, err := lc.KubeClient.CoreV1().Services(namespace).Get(context.Background(), svcName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	for _, p := range svc.Spec.Ports {
		if p.NodePort != 0 {
			return p.NodePort, nil
		}
	}
	return 0, nil
}
