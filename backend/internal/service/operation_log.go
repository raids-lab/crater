package service

import (
	"context"
	"time"

	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/internal/util"

	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

type OperationLogService struct {
	db *gorm.DB
}

func NewOperationLogService(db *gorm.DB) *OperationLogService {
	return &OperationLogService{db: db}
}

func (s *OperationLogService) database() *gorm.DB {
	if s.db != nil {
		return s.db
	}
	return query.GetDB()
}

var OpLog = &OperationLogService{}

func (s *OperationLogService) Create(
	ctx context.Context,
	operator, role, opType, target string,
	details datatypes.JSON,
	status, message string,
) error {
	log := &model.OperationLog{
		Operator:      operator,
		OperatorRole:  role,
		OperationType: opType,
		Target:        target,
		Details:       details,
		Status:        status,
		Message:       message,
	}
	return s.database().WithContext(ctx).Create(log).Error
}

func (s *OperationLogService) List(
	ctx context.Context, page, pageSize int, operator, opType, target, search string,
	startTime, endTime *time.Time, sortFields ...field.Expr,
) ([]*model.OperationLog, int64, error) {
	// OperationLog has no generated DAO; use GORM with bound field expressions.
	operatorField := field.NewString("operation_logs", "operator")
	typeField := field.NewString("operation_logs", "operation_type")
	targetField := field.NewString("operation_logs", "target")
	messageField := field.NewString("operation_logs", "message")
	createdAt := field.NewTime("operation_logs", "created_at")
	db := s.database().WithContext(ctx).Model(&model.OperationLog{})
	if operator != "" {
		db = db.Where(operatorField.Lower().Like(util.ContainsPattern(operator)).RawExpr())
	}
	if opType != "" {
		db = db.Where(typeField.Eq(opType).RawExpr())
	}
	if target != "" {
		db = db.Where(targetField.Lower().Like(util.ContainsPattern(target)).RawExpr())
	}
	if search != "" {
		pattern := util.ContainsPattern(search)
		db = db.Where(field.Or(
			operatorField.Lower().Like(pattern), typeField.Lower().Like(pattern),
			targetField.Lower().Like(pattern), messageField.Lower().Like(pattern),
		).RawExpr())
	}
	if startTime != nil {
		db = db.Where(createdAt.Gte(*startTime).RawExpr())
	}
	if endTime != nil {
		db = db.Where(createdAt.Lte(*endTime).RawExpr())
	}
	if len(sortFields) == 0 {
		sortFields = []field.Expr{createdAt.Desc(), field.NewUint("operation_logs", "id").Desc()}
	}
	orders := make([]clause.Expression, 0, len(sortFields))
	for _, order := range sortFields {
		orders = append(orders, order)
	}
	db = db.Clauses(clause.OrderBy{Expression: clause.CommaExpression{Exprs: orders}})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if int64(offset) >= total {
		return nil, total, nil
	}
	var logs []*model.OperationLog
	err := db.Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (s *OperationLogService) Clear(ctx context.Context) error {
	return s.database().WithContext(ctx).Unscoped().Where("1 = 1").Delete(&model.OperationLog{}).Error
}
