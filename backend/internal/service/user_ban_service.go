package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

type UserBanCapability string

const (
	UserBanCapabilityPlatformAccess  UserBanCapability = "platform_access"
	UserBanCapabilityJobSubmission   UserBanCapability = "job_submission"
	UserBanCapabilityImageBuild      UserBanCapability = "image_build"
	UserBanCapabilityModelDownload   UserBanCapability = "model_download"
	UserBanCapabilityDatasetDownload UserBanCapability = "dataset_download"
)

type UserBanPolicy struct {
	AllowPlatformAccess  bool `json:"allowPlatformAccess"`
	AllowJobSubmission   bool `json:"allowJobSubmission"`
	AllowImageBuild      bool `json:"allowImageBuild"`
	AllowModelDownload   bool `json:"allowModelDownload"`
	AllowDatasetDownload bool `json:"allowDatasetDownload"`
}

func (p UserBanPolicy) Allows(capability UserBanCapability) bool {
	switch capability {
	case UserBanCapabilityPlatformAccess:
		return p.AllowPlatformAccess
	case UserBanCapabilityJobSubmission:
		return p.AllowJobSubmission
	case UserBanCapabilityImageBuild:
		return p.AllowImageBuild
	case UserBanCapabilityModelDownload:
		return p.AllowModelDownload
	case UserBanCapabilityDatasetDownload:
		return p.AllowDatasetDownload
	default:
		return false
	}
}

type UserBanService struct {
	q *query.Query
}

func NewUserBanService(q *query.Query) *UserBanService {
	return &UserBanService{q: q}
}

var userBanConfigKeys = []string{
	model.ConfigKeyBanAllowPlatformAccess,
	model.ConfigKeyBanAllowJobSubmission,
	model.ConfigKeyBanAllowImageBuild,
	model.ConfigKeyBanAllowModelDownload,
	model.ConfigKeyBanAllowDatasetDownload,
}

func (s *UserBanService) GetPolicy(ctx context.Context) (UserBanPolicy, error) {
	policy := UserBanPolicy{}
	sc := s.q.SystemConfig
	configs, err := sc.WithContext(ctx).Where(sc.Key.In(userBanConfigKeys...)).Find()
	if err != nil {
		return policy, bizerr.Internal.DatabaseError.Wrap(err, "read user ban policy failed")
	}

	values := make(map[string]bool, len(configs))
	for _, config := range configs {
		value, err := strconv.ParseBool(config.Value)
		if err != nil {
			return policy, bizerr.Internal.DatabaseError.Wrap(err, "invalid user ban policy value")
		}
		values[config.Key] = value
	}

	policy.AllowPlatformAccess = values[model.ConfigKeyBanAllowPlatformAccess]
	policy.AllowJobSubmission = values[model.ConfigKeyBanAllowJobSubmission]
	policy.AllowImageBuild = values[model.ConfigKeyBanAllowImageBuild]
	policy.AllowModelDownload = values[model.ConfigKeyBanAllowModelDownload]
	policy.AllowDatasetDownload = values[model.ConfigKeyBanAllowDatasetDownload]
	return policy, nil
}

func (s *UserBanService) UpdatePolicy(ctx context.Context, policy UserBanPolicy) error {
	updates := map[string]string{
		model.ConfigKeyBanAllowPlatformAccess:  strconv.FormatBool(policy.AllowPlatformAccess),
		model.ConfigKeyBanAllowJobSubmission:   strconv.FormatBool(policy.AllowJobSubmission),
		model.ConfigKeyBanAllowImageBuild:      strconv.FormatBool(policy.AllowImageBuild),
		model.ConfigKeyBanAllowModelDownload:   strconv.FormatBool(policy.AllowModelDownload),
		model.ConfigKeyBanAllowDatasetDownload: strconv.FormatBool(policy.AllowDatasetDownload),
	}
	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return applySystemConfigUpdates(ctx, tx, updates)
	})
}

func (s *UserBanService) RequireCapability(
	ctx context.Context,
	userID uint,
	capability UserBanCapability,
) error {
	u := s.q.User
	user, err := u.WithContext(ctx).
		Select(u.ID, u.BannedAt).
		Where(u.ID.Eq(userID), u.DeletedAt.IsNull()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerr.Auth.TokenInvalid.New("User not found")
		}
		return bizerr.Internal.DatabaseError.Wrap(err, "check user ban status failed")
	}
	if user.BannedAt == nil {
		return nil
	}

	policy, err := s.GetPolicy(ctx)
	if err != nil {
		return err
	}
	if policy.Allows(capability) {
		return nil
	}
	return bizerr.Forbidden.UserBanned.New(userBanCapabilityMessage(capability))
}

func userBanCapabilityMessage(capability UserBanCapability) string {
	switch capability {
	case UserBanCapabilityPlatformAccess:
		return "Your account is banned from accessing the platform. Contact a platform administrator."
	case UserBanCapabilityJobSubmission:
		return "Your account is banned from submitting new jobs. Existing jobs remain available."
	case UserBanCapabilityImageBuild:
		return "Your account is banned from building images."
	case UserBanCapabilityModelDownload:
		return "Your account is banned from downloading models."
	case UserBanCapabilityDatasetDownload:
		return "Your account is banned from downloading datasets."
	default:
		return "Your account is banned from performing this operation."
	}
}

func (s *UserBanService) SetBan(
	ctx context.Context,
	username string,
	operatorID uint,
	operatorName string,
	banned bool,
	reason string,
) (*model.User, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, bizerr.BadRequest.ParameterError.New("Ban reason is required")
	}

	var result model.User
	err := query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("name = ? AND deleted_at IS NULL", username).
			First(&result).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return bizerr.NotFound.DataBaseNotFound.New("User not found")
			}
			return bizerr.Internal.DatabaseError.Wrap(err, "load user for ban update failed")
		}
		if result.ID == operatorID {
			return bizerr.Forbidden.PermissionDenied.New("Administrators cannot ban or unban themselves")
		}
		if (result.BannedAt != nil) == banned {
			state := "unbanned"
			if banned {
				state = "banned"
			}
			return bizerr.Conflict.ResourceStatusError.New(fmt.Sprintf("User is already %s", state))
		}

		now := time.Now()
		var bannedAt *time.Time
		action := model.UserBanActionUnban
		if banned {
			bannedAt = &now
			action = model.UserBanActionBan
		}
		if err := tx.Model(&model.User{}).
			Where("id = ?", result.ID).
			Update("banned_at", bannedAt).Error; err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "update user ban state failed")
		}
		if err := tx.Create(&model.UserBanRecord{
			UserID:       result.ID,
			UserName:     result.Name,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			Action:       action,
			Reason:       reason,
		}).Error; err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "create user ban record failed")
		}
		result.BannedAt = bannedAt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *UserBanService) ListRecords(ctx context.Context, username string) (*model.User, []*model.UserBanRecord, error) {
	u := s.q.User
	user, err := u.WithContext(ctx).
		Where(u.Name.Eq(username), u.DeletedAt.IsNull()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, bizerr.NotFound.DataBaseNotFound.New("User not found")
		}
		return nil, nil, bizerr.Internal.DatabaseError.Wrap(err, "load user ban status failed")
	}

	recordsQuery := s.q.UserBanRecord
	records, err := recordsQuery.WithContext(ctx).
		Where(recordsQuery.UserID.Eq(user.ID)).
		Order(recordsQuery.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, nil, bizerr.Internal.DatabaseError.Wrap(err, "list user ban records failed")
	}
	return user, records, nil
}
