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

	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/utils"
)

// ApplyUserJobCapabilities reads authorization at creation time, never from a token or template.
// Call only for ordinary user jobs, before persisting or submitting their pod templates.
func ApplyUserJobCapabilities(ctx context.Context, userID uint, pods ...*corev1.PodSpec) error {
	u := query.User
	user, err := u.WithContext(ctx).Select(u.ID, u.ContainerSysAdminEnabled).Where(u.ID.Eq(userID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.Auth.TokenInvalid.New("User not found")
	}
	if err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "Unable to read job container permissions")
	}
	for _, pod := range pods {
		utils.AddJobProfilingCapabilities(pod)
		utils.SetJobSysAdminCapability(pod, user.ContainerSysAdminEnabled)
	}
	return nil
}

// AdminUpdateUserAttributes updates only the intended columns in one statement.
// An omitted grant preserves authorization for older clients editing personal information.
func AdminUpdateUserAttributes(ctx context.Context, username string, attributes *model.UserAttribute, enabled *bool) error {
	updates := map[string]any{
		"attributes": datatypes.NewJSONType(*attributes),
		"nickname":   attributes.Nickname,
	}
	if enabled != nil {
		updates["container_sys_admin_enabled"] = *enabled
	}
	u := query.User
	result, err := u.WithContext(ctx).Where(u.Name.Eq(username)).Updates(updates)
	if err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "Unable to update user attributes")
	}
	if result.RowsAffected == 0 {
		return bizerr.NotFound.DataBaseNotFound.New("User not found")
	}
	return nil
}
