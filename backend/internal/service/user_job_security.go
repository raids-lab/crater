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

// UserJobPermissions describes capabilities a user may explicitly request for a new job.
type UserJobPermissions struct {
	ContainerSysAdminEnabled bool `json:"containerSysAdminEnabled"`
}

// GetUserJobPermissions reads current authorization, never from a token or template.
func GetUserJobPermissions(ctx context.Context, userID uint) (*UserJobPermissions, error) {
	u := query.User
	user, err := u.WithContext(ctx).Select(u.ID, u.ContainerSysAdminEnabled).Where(u.ID.Eq(userID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.Auth.TokenInvalid.New("User not found")
	}
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "Unable to read job container permissions")
	}
	return &UserJobPermissions{ContainerSysAdminEnabled: user.ContainerSysAdminEnabled}, nil
}

// ApplyUserJobCapabilities requires both current authorization and explicit per-job consent.
// Call only for ordinary user jobs, before persisting or submitting their pod templates.
func ApplyUserJobCapabilities(ctx context.Context, userID uint, sysAdminEnabled bool, pods ...*corev1.PodSpec) error {
	permissions, err := GetUserJobPermissions(ctx, userID)
	if err != nil {
		return err
	}
	if sysAdminEnabled && !permissions.ContainerSysAdminEnabled {
		return bizerr.Forbidden.PermissionDenied.New("You are not authorized to enable SYS_ADMIN for job containers")
	}
	for _, pod := range pods {
		utils.AddJobProfilingCapabilities(pod)
		utils.SetJobSysAdminCapability(pod, sysAdminEnabled)
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
