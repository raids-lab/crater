package quota

import (
	"context"

	"github.com/go-logr/logr"
	aijobapi "k8s.io/ai-task-controller/pkg/apis/aijob/v1alpha1"
	quotadb "k8s.io/ai-task-controller/pkg/db/quota"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	quotaDB = quotadb.NewDBService()
)

type QuotaController struct {
	// common
	client.Client // for getting aijobs
	Scheme        *runtime.Scheme
	Log           logr.Logger
}

func NewQuotaController() *QuotaController {
	return &QuotaController{}
}

func (qc *QuotaController) Start() error {
	return nil
}

// 获取所有数据库的quota数据，更新到quotaInfo
func (qc *QuotaController) updateQuotaInfos() error {
	// 1. get all quota in db
	quotaList, err := quotaDB.ListAllQuotas()
	if err != nil {
		return err
	}
	for _, quota := range quotaList {
		// 2. add or update quota hard
		_, quotaInfo := QuotaInfosData.AddOrUpdateQuotaInfo(quota.UserName, quota)

		// 3. get all aijobs for this quota
		aijobList, err := qc.listAIJobsForNamespace(quota.UserName)
		if err != nil {
			// todo: handle err
		}
		// 4. add aijobs to quotaInfo
		for _, aijob := range aijobList.Items {
			quotaInfo.AddJob(&aijob)
		}
	}
	return nil
}

func (qc *QuotaController) listAIJobsForNamespace(namespace string) (*aijobapi.AIJobList, error) {
	var aijobList *aijobapi.AIJobList
	err := qc.Client.List(context.Background(), aijobList, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}
	return aijobList, nil
}
