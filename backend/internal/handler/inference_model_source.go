package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

func loadAccessibleModelDataset(ctx context.Context, datasetID uint, token util.JWTMessage) (*model.Dataset, error) {
	d := query.Dataset
	dataset, err := d.WithContext(ctx).Where(d.ID.Eq(datasetID), d.Type.Eq(string(model.DataTypeModel))).First()
	if err != nil {
		return nil, bizerr.NotFound.DataBaseNotFound.New("selected platform model was not found")
	}
	if dataset.UserID == token.UserID {
		return dataset, nil
	}
	ud := query.UserDataset
	if _, err := ud.WithContext(ctx).Where(ud.UserID.Eq(token.UserID), ud.DatasetID.Eq(datasetID)).First(); err == nil {
		return dataset, nil
	}
	qd := query.AccountDataset
	if _, err := qd.WithContext(ctx).Where(qd.AccountID.Eq(token.AccountID), qd.DatasetID.Eq(datasetID)).First(); err == nil {
		return dataset, nil
	}
	return nil, bizerr.Forbidden.PermissionDenied.New("you do not have permission to use the selected platform model")
}

func datasetToKthenaModelURI(dataset *model.Dataset) string {
	url := strings.TrimSpace(dataset.URL)
	if strings.HasPrefix(url, "pvc://") {
		return url
	}
	return "pvc://" + datasetModelCacheMountPath() + "/" + strings.TrimLeft(url, "/")
}

func datasetModelCacheURI() string {
	pvcName := strings.TrimSpace(config.GetConfig().Storage.PVC.ReadWriteMany)
	if pvcName == "" {
		return ""
	}
	return "pvc://" + pvcName
}

func datasetModelCacheMountPath() string {
	pvcName := strings.TrimSpace(config.GetConfig().Storage.PVC.ReadWriteMany)
	if pvcName == "" {
		return ""
	}
	return "/" + strings.Trim(pvcName, "/")
}

func inferServedModelName(modelURI string) string {
	if strings.Contains(modelURI, "://") {
		parts := strings.Split(modelURI, "://")
		modelURI = parts[len(parts)-1]
	}
	modelURI = strings.TrimSuffix(modelURI, "/")
	parts := strings.Split(modelURI, "/")
	if len(parts) == 0 {
		return modelURI
	}
	return parts[len(parts)-1]
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func nonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
