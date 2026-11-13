package handler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/internal/util"
)

type ApproveJobOrderReq struct {
	Lock        *vcjobservice.JobLockDuration `json:"lock"`
	ReviewNotes string                        `json:"reviewNotes"`
}

type ApproveJobOrderResp struct {
	Message      string                `json:"message"`
	LockMessage  string                `json:"lockMessage"`
	ApprovedLock model.ApprovedJobLock `json:"approvedLock"`
}

// ApproveJobOrderAdmin godoc
//
//	@Summary		批准作业工单并锁定作业
//	@Description	在同一事务中锁定运行中的作业、批准工单并记录实际批准时长，省略锁定参数时使用工单申请时长
//	@Tags			approvalorder
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path	int					true	"工单ID"
//	@Param			body	body	ApproveJobOrderReq	true	"锁定时长及审核备注"
//	@Success		200	{object}	resputil.Response[ApproveJobOrderResp]
//	@Failure		400	{object}	resputil.Response[any]
//	@Failure		403	{object}	resputil.Response[any]
//	@Failure		404	{object}	resputil.Response[any]
//	@Failure		409	{object}	resputil.Response[any]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/admin/approvalorder/{id}/approve-and-lock [put]
func (mgr *ApprovalOrderMgr) ApproveJobOrderAdmin(c *gin.Context) {
	var orderID ApprovalOrderIDReq
	if err := c.ShouldBindUri(&orderID); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid approval order ID"))
		return
	}
	var req ApproveJobOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid approval lock request"))
		return
	}
	token := util.GetToken(c)
	if token.UserID == 0 {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("cannot get user id"))
		return
	}
	if token.RolePlatform != model.RoleAdmin {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("only admins can approve job orders"))
		return
	}

	var result ApproveJobOrderResp
	err := query.Q.Transaction(func(tx *query.Query) error {
		var err error
		result, err = approvePendingJobOrder(c, tx, orderID.ID, token.UserID, req)
		return err
	})
	if err != nil {
		if _, ok := bizerr.FromError(err); !ok {
			err = bizerr.Internal.DatabaseError.Wrap(err, "failed to commit job approval")
		}
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, result)
}

func approvePendingJobOrder(
	ctx context.Context, tx *query.Query, orderID, reviewerID uint, req ApproveJobOrderReq,
) (ApproveJobOrderResp, error) {
	var result ApproveJobOrderResp
	ao := tx.ApprovalOrder
	orders := ao.WithContext(ctx).Where(ao.ID.Eq(orderID))
	order, err := orders.Clauses(clause.Locking{Strength: "UPDATE"}).Take()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, bizerr.NotFound.DataBaseNotFound.New("approval order not found")
	}
	if err != nil {
		return result, bizerr.Internal.DatabaseError.Wrap(err, "failed to load approval order")
	}
	if order.Type != model.ApprovalOrderTypeJob {
		return result, bizerr.BadRequest.ParameterError.New("only job orders can be approved and locked")
	}
	if order.Status != model.ApprovalOrderStatusPending {
		return result, bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be reviewed")
	}
	content := order.Content.Data()
	duration := req.Lock
	if duration == nil {
		duration, err = requestedJobLock(content.ApprovalOrderExtensionHours)
		if err != nil {
			return result, err
		}
	}
	lock, err := vcjobservice.ExtendJobLock(ctx, tx, order.Name, *duration, true)
	if err != nil {
		return result, err
	}
	content.ApprovedLock = lock
	info, err := ao.WithContext(ctx).
		Where(ao.ID.Eq(orderID), ao.Status.Eq(string(model.ApprovalOrderStatusPending))).
		Updates(map[string]any{
			"content": datatypes.NewJSONType(content), "status": model.ApprovalOrderStatusApproved,
			"reviewer_id": reviewerID, "review_notes": req.ReviewNotes,
		})
	if err != nil {
		return result, bizerr.Internal.DatabaseError.Wrap(err, "failed to record job approval")
	}
	if info.RowsAffected == 0 {
		return result, bizerr.Conflict.ResourceStatusError.New("only pending approval orders can be reviewed")
	}
	result.Message = "approval order approved"
	result.LockMessage = fmt.Sprintf("locked %s until %s", order.Name, lock.LockedUntil.Format(time.RFC3339))
	if lock.IsPermanent {
		result.LockMessage = fmt.Sprintf("permanently locked %s", order.Name)
	}
	result.ApprovedLock = *lock
	return result, nil
}

func requestedJobLock(hours uint) (*vcjobservice.JobLockDuration, error) {
	if hours == 0 {
		return nil, bizerr.BadRequest.MissingParameter.New("provide a positive lock duration or request a permanent lock")
	}
	if hours > uint(math.MaxInt64/int64(time.Hour)) {
		return nil, bizerr.BadRequest.ParameterError.New("requested lock duration is too large")
	}
	return &vcjobservice.JobLockDuration{Hours: int64(hours)}, nil
}

// A failed automatic lock leaves the order pending. A failed insert rolls the lock back.
func createAutoApprovedJobOrder(ctx context.Context, order *model.ApprovalOrder) (bool, error) {
	content := order.Content.Data()
	duration, err := requestedJobLock(content.ApprovalOrderExtensionHours)
	if err != nil {
		return false, err
	}
	var lockFailed bool
	err = query.Q.Transaction(func(tx *query.Query) error {
		lock, err := vcjobservice.ExtendJobLock(ctx, tx, order.Name, *duration, false)
		if err != nil {
			lockFailed = true
			return err
		}
		content.ApprovedLock = lock
		order.Content = datatypes.NewJSONType(content)
		order.Status = model.ApprovalOrderStatusApproved
		order.ReviewNotes = autoApprovalReason
		return tx.ApprovalOrder.WithContext(ctx).Create(order)
	})
	if lockFailed {
		klog.Warningf("automatic approval could not lock job %s: %v", order.Name, err)
		return false, nil
	}
	return err == nil, err
}
