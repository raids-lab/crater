package crclient

import (
	"context"

	recommenddljobapi "github.com/raids-lab/crater/pkg/apis/recommenddljob/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DataSetClient struct {
	client.Client
}

func (c *DataSetClient) ListDataSets(ctx context.Context, namespace string) ([]*recommenddljobapi.DataSet, error) {
	dataSetList := &recommenddljobapi.DataSetList{}
	if err := c.List(ctx, dataSetList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}
	ret := make([]*recommenddljobapi.DataSet, 0, len(dataSetList.Items))
	for i := range dataSetList.Items {
		ret = append(ret, &dataSetList.Items[i])
	}
	return ret, nil
}

func (c *DataSetClient) GetDataSet(ctx context.Context, name, namespace string) (dataset *recommenddljobapi.DataSet, err error) {
	dataset = &recommenddljobapi.DataSet{}
	if err = c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, dataset); err != nil {
		return nil, err
	}
	return
}
