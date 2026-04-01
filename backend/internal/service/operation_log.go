package service

import (
	"context"

	"gorm.io/datatypes"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func init() {
	if err := query.GetDB().AutoMigrate(&model.OperationLog{}); err != nil {
		klog.Fatalf("auto migrate operation_logs failed: %v", err)
	}
}

type OperationLogService struct{}

var OpLog = &OperationLogService{}

func (s *OperationLogService) Create(ctx context.Context, operator, role, opType, target string, details datatypes.JSON, status, message string) error {
	log := &model.OperationLog{
		Operator:      operator,
		OperatorRole:  role,
		OperationType: opType,
		Target:        target,
		Details:       details,
		Status:        status,
		Message:       message,
	}
	return query.GetDB().Create(log).Error
}

func (s *OperationLogService) List(ctx context.Context, page, pageSize int, operator, opType, target, search string) ([]*model.OperationLog, int64, error) {
	var logs []*model.OperationLog
	var total int64
	db := query.GetDB().Model(&model.OperationLog{})

	if operator != "" {
		db = db.Where("operator LIKE ?", "%"+operator+"%")
	}
	if opType != "" {
		db = db.Where("operation_type = ?", opType)
	}
	if target != "" {
		db = db.Where("target LIKE ?", "%"+target+"%")
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		db = db.Where("operator LIKE ? OR operation_type LIKE ? OR target LIKE ? OR message LIKE ?", searchPattern, searchPattern, searchPattern, searchPattern)
	}

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (s *OperationLogService) Clear(ctx context.Context) error {
	return query.GetDB().WithContext(ctx).Unscoped().Where("1 = 1").Delete(&model.OperationLog{}).Error
}
