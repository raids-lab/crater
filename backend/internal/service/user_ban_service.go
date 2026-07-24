package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/utils"
)

type UserBanCapability string

const (
	UserBanCapabilityPlatformAccess  UserBanCapability = "platform_access"
	UserBanCapabilityJobSubmission   UserBanCapability = "job_submission"
	UserBanCapabilityImageBuild      UserBanCapability = "image_build"
	UserBanCapabilityModelDownload   UserBanCapability = "model_download"
	UserBanCapabilityDatasetDownload UserBanCapability = "dataset_download"
)

func isUserBanCapabilityRestricted(
	restrictions model.UserBanRestrictions,
	capability UserBanCapability,
) bool {
	switch capability {
	case UserBanCapabilityPlatformAccess:
		return restrictions.PlatformAccess
	case UserBanCapabilityJobSubmission:
		return restrictions.JobSubmission
	case UserBanCapabilityImageBuild:
		return restrictions.ImageBuild
	case UserBanCapabilityModelDownload:
		return restrictions.ModelDownload
	case UserBanCapabilityDatasetDownload:
		return restrictions.DatasetDownload
	default:
		return true
	}
}

type UserBanService struct {
	q *query.Query
}

func NewUserBanService(q *query.Query) *UserBanService {
	return &UserBanService{q: q}
}

func (s *UserBanService) RequireCapability(
	ctx context.Context,
	userID uint,
	capability UserBanCapability,
) error {
	u := s.q.User
	user, err := u.WithContext(ctx).
		Select(u.ID, u.BannedTimestamp, u.BanRestrictions).
		Where(u.ID.Eq(userID), u.DeletedAt.IsNull()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerr.Auth.TokenInvalid.New("User not found")
		}
		return bizerr.Internal.DatabaseError.Wrap(err, "check user ban status failed")
	}
	if !isUserBanCapabilityBlockedAt(
		user.BannedTimestamp,
		user.BanRestrictions.Data(),
		capability,
		utils.GetLocalTime(),
	) {
		return nil
	}
	return bizerr.Forbidden.UserBanned.New(userBanCapabilityMessage(capability))
}

func isUserBanCapabilityBlockedAt(
	bannedTimestamp *time.Time,
	restrictions model.UserBanRestrictions,
	capability UserBanCapability,
	now time.Time,
) bool {
	return isUserBannedAt(bannedTimestamp, now) &&
		isUserBanCapabilityRestricted(restrictions, capability)
}

func IsUserBanned(bannedTimestamp *time.Time) bool {
	return isUserBannedAt(bannedTimestamp, utils.GetLocalTime())
}

func isUserBannedAt(bannedTimestamp *time.Time, now time.Time) bool {
	return bannedTimestamp != nil && bannedTimestamp.After(now)
}

func IsUserPermanentlyBanned(bannedTimestamp *time.Time) bool {
	return bannedTimestamp != nil && utils.IsPermanentTime(*bannedTimestamp)
}

// EffectiveUserBanRestrictions returns only restrictions that are currently active.
// Stored restrictions may remain after expiry; the timestamp is the sole source of ban status.
func EffectiveUserBanRestrictions(
	banned bool,
	restrictions model.UserBanRestrictions,
) model.UserBanRestrictions {
	if !banned {
		return model.UserBanRestrictions{}
	}
	return restrictions
}

func nextUserBanTimestamp(
	current *time.Time,
	now time.Time,
	isPermanent bool,
	duration time.Duration,
) (*time.Time, model.UserBanAction) {
	action := model.UserBanActionBan
	base := now
	if isUserBannedAt(current, now) {
		action = model.UserBanActionExtend
		base = *current
	}
	if isPermanent {
		permanent := utils.GetPermanentTime()
		return &permanent, action
	}
	until := base.Add(duration)
	return &until, action
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

func validateUserBanRequest(
	banned bool,
	isPermanent bool,
	duration time.Duration,
	restrictions model.UserBanRestrictions,
	reason string,
) error {
	if banned && reason == "" {
		return bizerr.BadRequest.ParameterError.New("Ban reason is required")
	}
	if banned && !isPermanent && duration <= 0 {
		return bizerr.BadRequest.ParameterError.New("Ban duration must be positive unless permanent ban is requested")
	}
	if banned && !restrictions.Any() {
		return bizerr.BadRequest.ParameterError.New("At least one ban restriction is required")
	}
	return nil
}

func (s *UserBanService) SetBan(
	ctx context.Context,
	username string,
	operatorID uint,
	operatorName string,
	banned bool,
	isPermanent bool,
	duration time.Duration,
	restrictions model.UserBanRestrictions,
	reason string,
) (*model.User, error) {
	reason = strings.TrimSpace(reason)
	if err := validateUserBanRequest(banned, isPermanent, duration, restrictions, reason); err != nil {
		return nil, err
	}
	if !banned {
		restrictions = model.UserBanRestrictions{}
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
		now := utils.GetLocalTime()
		var bannedTimestamp *time.Time
		action := model.UserBanActionUnban
		if banned {
			if IsUserPermanentlyBanned(result.BannedTimestamp) {
				return bizerr.Conflict.ResourceStatusError.New("User is already permanently banned")
			}
			bannedTimestamp, action = nextUserBanTimestamp(
				result.BannedTimestamp,
				now,
				isPermanent,
				duration,
			)
		}
		encodedRestrictions := datatypes.NewJSONType(restrictions)
		if err := tx.Model(&model.User{}).
			Where("id = ?", result.ID).
			Updates(map[string]any{
				"banned_timestamp": bannedTimestamp,
				"ban_restrictions": encodedRestrictions,
			}).Error; err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "update user ban state failed")
		}
		if err := tx.Create(&model.UserBanRecord{
			UserID:          result.ID,
			UserName:        result.Name,
			OperatorID:      operatorID,
			OperatorName:    operatorName,
			Action:          action,
			BannedTimestamp: bannedTimestamp,
			BanRestrictions: encodedRestrictions,
			Reason:          reason,
		}).Error; err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "create user ban record failed")
		}
		result.BannedTimestamp = bannedTimestamp
		result.BanRestrictions = encodedRestrictions
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *UserBanService) GetCurrentState(
	ctx context.Context,
	userID uint,
) (*model.User, *model.UserBanRecord, error) {
	u := s.q.User
	user, err := u.WithContext(ctx).
		Where(u.ID.Eq(userID), u.DeletedAt.IsNull()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, bizerr.Auth.TokenInvalid.New("User not found")
		}
		return nil, nil, bizerr.Internal.DatabaseError.Wrap(err, "load current user ban state failed")
	}
	if !IsUserBanned(user.BannedTimestamp) {
		return user, nil, nil
	}

	recordsQuery := s.q.UserBanRecord
	record, err := recordsQuery.WithContext(ctx).
		Where(recordsQuery.UserID.Eq(user.ID)).
		Order(recordsQuery.CreatedAt.Desc()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, nil, nil
		}
		return nil, nil, bizerr.Internal.DatabaseError.Wrap(err, "load current user ban record failed")
	}
	return user, record, nil
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
