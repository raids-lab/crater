// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

type ImageBuildDetailService struct {
	q *query.Query
}

func NewImageBuildDetailService(q *query.Query) *ImageBuildDetailService {
	return &ImageBuildDetailService{q: q}
}

// Get authorizes build configuration reads independently of build management.
// admin must only be set by a handler registered on the Admin route group.
func (s *ImageBuildDetailService) Get(
	ctx context.Context, name string, userID, accountID uint, admin bool,
) (*model.Kaniko, bool, error) {
	k := s.q.Kaniko
	build, err := k.WithContext(ctx).Where(k.ImagePackName.Eq(name)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, bizerr.NotFound.DataBaseNotFound.New("Image build not found")
		}
		return nil, false, bizerr.Internal.DatabaseError.Wrap(err, "Unable to read image build")
	}
	if admin || build.UserID == userID {
		return build, true, nil
	}
	i := s.q.Image
	image, err := i.WithContext(ctx).Where(i.ImagePackName.Eq(name)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, bizerr.Forbidden.PermissionDenied.New("You do not have permission to view this image build")
		}
		return nil, false, bizerr.Internal.DatabaseError.Wrap(err, "Unable to read image visibility")
	}
	if image.IsPublic {
		return build, false, nil
	}
	a := s.q.ImageAccount
	accounts := []uint{model.DefaultAccountID}
	if accountID != 0 {
		accounts = append(accounts, accountID)
	}
	count, err := a.WithContext(ctx).Where(a.ImageID.Eq(image.ID), a.AccountID.In(accounts...)).Count()
	if err != nil {
		return nil, false, bizerr.Internal.DatabaseError.Wrap(err, "Unable to read image account access")
	}
	if count > 0 {
		return build, false, nil
	}
	u := s.q.ImageUser
	count, err = u.WithContext(ctx).Where(u.ImageID.Eq(image.ID), u.UserID.Eq(userID)).Count()
	if err != nil {
		return nil, false, bizerr.Internal.DatabaseError.Wrap(err, "Unable to read image user access")
	}
	if count > 0 {
		return build, false, nil
	}
	return nil, false, bizerr.Forbidden.PermissionDenied.New("You do not have permission to view this image build")
}
