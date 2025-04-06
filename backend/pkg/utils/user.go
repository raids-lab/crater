package utils

import (
	"context"
	"time"

	"github.com/raids-lab/crater/dao/query"
)

func IsUserEmailVerified(c context.Context, userID uint) bool {
	u := query.User
	user, err := u.WithContext(c).
		Where(u.ID.Eq(userID)).
		First()
	if err != nil {
		return false
	}
	return IsEmailVerified(user.LastEmailVerifiedAt)
}

func IsEmailVerified(lastEmailVerifiedAt time.Time) bool {
	// todo: emailValidityDays写到配置文件中
	emailValidityDays := 180

	if lastEmailVerifiedAt.IsZero() {
		return false
	}

	curTime := GetLocalTime()

	return lastEmailVerifiedAt.AddDate(0, 0, emailValidityDays).After(curTime)
}
