package image

import (
	"context"
	"sort"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/util"
)

//nolint:gochecknoinits // The agent package discovers this facade through handler registration.
func init() {
	handler.RegisterImageInsightReaderFactory(NewAgentImageInsightReader)
}

type agentImageInsightReader struct{}

func NewAgentImageInsightReader(_ *handler.RegisterConfig) handler.ImageInsightReader {
	return &agentImageInsightReader{}
}

func (r *agentImageInsightReader) ListAccessibleImages(
	ctx context.Context,
	token util.JWTMessage,
) ([]handler.ImageAccessRecord, error) {
	if token.RolePlatform == model.RoleAdmin {
		return r.listAdminImages(ctx)
	}
	return r.listUserImages(ctx, token)
}

func (r *agentImageInsightReader) listAdminImages(ctx context.Context) ([]handler.ImageAccessRecord, error) {
	publicImageIDs, err := r.listPublicImageIDs(ctx)
	if err != nil {
		return nil, err
	}
	publicIDSet := make(map[uint]struct{}, len(publicImageIDs))
	for _, id := range publicImageIDs {
		publicIDSet[id] = struct{}{}
	}

	iq := query.Image
	images, err := iq.WithContext(ctx).Preload(iq.User).Order(iq.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, err
	}

	records := make([]handler.ImageAccessRecord, 0, len(images))
	for _, item := range images {
		status := model.Private
		if _, ok := publicIDSet[item.ID]; ok || item.IsPublic {
			status = model.Public
		}
		records = append(records, handler.ImageAccessRecord{
			Image:       item,
			ShareStatus: status,
		})
	}
	return records, nil
}

func (r *agentImageInsightReader) listUserImages(
	ctx context.Context,
	token util.JWTMessage,
) ([]handler.ImageAccessRecord, error) {
	results := make([]handler.ImageAccessRecord, 0)
	seen := make(map[uint]struct{})
	appendUnique := func(images []*model.Image, shareStatus model.ImageShareType) {
		for _, imageRecord := range images {
			if imageRecord == nil || imageRecord.ID == 0 || imageRecord.ImageLink == "" {
				continue
			}
			if _, ok := seen[imageRecord.ID]; ok {
				continue
			}
			seen[imageRecord.ID] = struct{}{}
			results = append(results, handler.ImageAccessRecord{
				Image:       imageRecord,
				ShareStatus: shareStatus,
			})
		}
	}

	iq := query.Image
	oldPublicImages, err := iq.WithContext(ctx).
		Preload(iq.User).
		Where(iq.IsPublic.Is(true)).
		Order(iq.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, err
	}
	appendUnique(oldPublicImages, model.Public)

	newPublicImages, err := r.listAccountSharedImages(ctx, model.DefaultAccountID)
	if err != nil {
		return nil, err
	}
	appendUnique(newPublicImages, model.Public)

	accountImages, err := r.listAccountSharedImages(ctx, token.AccountID)
	if err != nil {
		return nil, err
	}
	appendUnique(accountImages, model.AccountShare)

	privateImages, err := iq.WithContext(ctx).
		Preload(iq.User).
		Where(iq.UserID.Eq(token.UserID)).
		Order(iq.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, err
	}
	appendUnique(privateImages, model.Private)

	userImages, err := r.listUserSharedImages(ctx, token.UserID)
	if err != nil {
		return nil, err
	}
	appendUnique(userImages, model.UserShare)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Image.CreatedAt.After(results[j].Image.CreatedAt)
	})
	return results, nil
}

func (r *agentImageInsightReader) listPublicImageIDs(ctx context.Context) ([]uint, error) {
	imageAccountQuery := query.ImageAccount
	publicImageIDs := []uint{}
	if err := imageAccountQuery.WithContext(ctx).
		Where(imageAccountQuery.AccountID.Eq(model.DefaultAccountID)).
		Pluck(imageAccountQuery.ImageID, &publicImageIDs); err != nil {
		return nil, err
	}

	imageQuery := query.Image
	oldPublicImageIDs := []uint{}
	if err := imageQuery.WithContext(ctx).
		Where(imageQuery.IsPublic.Is(true)).
		Pluck(imageQuery.ID, &oldPublicImageIDs); err != nil {
		return nil, err
	}
	return append(publicImageIDs, oldPublicImageIDs...), nil
}

//nolint:dupl // GORM gen exposes separate typed query builders for account and user image shares.
func (r *agentImageInsightReader) listAccountSharedImages(ctx context.Context, accountID uint) ([]*model.Image, error) {
	imageShareQuery := query.ImageAccount
	imageShares, err := imageShareQuery.WithContext(ctx).
		Preload(imageShareQuery.Image).
		Preload(imageShareQuery.Image.User).
		Where(imageShareQuery.AccountID.Eq(accountID)).
		Find()
	if err != nil {
		return nil, err
	}
	return imagesFromShares(imageShares, func(imageShare *model.ImageAccount) *model.Image {
		return &imageShare.Image
	}), nil
}

//nolint:dupl // GORM gen exposes separate typed query builders for account and user image shares.
func (r *agentImageInsightReader) listUserSharedImages(ctx context.Context, userID uint) ([]*model.Image, error) {
	imageShareQuery := query.ImageUser
	imageShares, err := imageShareQuery.WithContext(ctx).
		Preload(imageShareQuery.Image).
		Preload(imageShareQuery.Image.User).
		Where(imageShareQuery.UserID.Eq(userID)).
		Find()
	if err != nil {
		return nil, err
	}
	return imagesFromShares(imageShares, func(imageShare *model.ImageUser) *model.Image {
		return &imageShare.Image
	}), nil
}

func imagesFromShares[T any](imageShares []T, imageOf func(T) *model.Image) []*model.Image {
	images := make([]*model.Image, 0, len(imageShares))
	for _, imageShare := range imageShares {
		images = append(images, imageOf(imageShare))
	}
	return images
}
