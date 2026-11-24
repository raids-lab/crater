package service

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
)

// PatchUserProfile atomically merges submitted keys and preserves unrelated edits.
func PatchUserProfile(ctx context.Context, username string, patch, columns map[string]any) error {
	return patchUserProfile(ctx, username, patch, columns, false)
}

// PatchLDAPUserProfile protects email using its current database verification state.
func PatchLDAPUserProfile(ctx context.Context, username string, patch map[string]any) error {
	return patchUserProfile(ctx, username, patch, nil, true)
}

func patchUserProfile(ctx context.Context, username string, patch, columns map[string]any, protectEmail bool) error {
	db := query.User.WithContext(ctx).UnderlyingDB().Model(&model.User{}).Where("name = ?", username)
	if columns == nil {
		columns = map[string]any{}
	}
	postgres := db.Name() == "postgres"
	base := any(gorm.Expr("COALESCE(NULLIF(attributes, 'null'), '{}')"))
	changed := false
	if nickname, ok := patch["nickname"]; ok {
		columns["nickname"] = nickname
		delete(patch, "nickname")
		base = nicknameExpression(postgres, nickname)
		changed = true
	}
	if email, ok := patch["email"]; ok && protectEmail {
		raw, err := json.Marshal(map[string]any{"email": email})
		if err != nil {
			return err
		}
		merged := mergeProfileExpression(postgres, base, string(raw))
		base = gorm.Expr("CASE WHEN last_email_verified_at IS NULL THEN ? ELSE ? END", merged, base)
		delete(patch, "email")
		changed = true
	}
	if len(patch) > 0 {
		raw, err := json.Marshal(patch)
		if err != nil {
			return err
		}
		base = mergeProfileExpression(postgres, base, string(raw))
		changed = true
	}
	if changed {
		columns["attributes"] = base
	}
	if len(columns) == 0 {
		return nil
	}
	result := db.Updates(columns)
	if result.Error != nil {
		return bizerr.Internal.DatabaseError.Wrap(result.Error, "Unable to update user profile")
	}
	if result.RowsAffected == 0 {
		return bizerr.NotFound.DataBaseNotFound.New("User not found")
	}
	return nil
}

func mergeProfileExpression(postgres bool, base any, raw string) any {
	if postgres {
		return gorm.Expr("COALESCE(NULLIF((?)::jsonb, 'null'::jsonb), '{}'::jsonb) || ?::jsonb", base, raw)
	}
	return gorm.Expr("json_patch(COALESCE(NULLIF(?, 'null'), '{}'), ?)", base, raw)
}

func nicknameExpression(postgres bool, nickname any) any {
	raw, _ := json.Marshal(nickname)
	if postgres {
		return gorm.Expr("CASE WHEN attributes IS NULL OR attributes::jsonb IN ('{}'::jsonb,'null'::jsonb) "+
			"THEN attributes ELSE attributes::jsonb || jsonb_build_object('nickname', ?::jsonb) END", string(raw))
	}
	return gorm.Expr("CASE WHEN attributes IS NULL OR attributes IN ('{}','null') THEN attributes "+
		"ELSE json_set(attributes, '$.nickname', json(?)) END", string(raw))
}
