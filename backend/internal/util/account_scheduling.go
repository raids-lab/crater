package util

import (
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
)

// AccountTolerations returns the toleration used by jobs in the selected account.
func AccountTolerations(token JWTMessage) []v1.Toleration {
	if token.AccountID == model.DefaultAccountID {
		return nil
	}
	return []v1.Toleration{{
		Key:      "crater.raids.io/account",
		Operator: v1.TolerationOpEqual,
		Value:    token.AccountName,
		Effect:   v1.TaintEffectNoSchedule,
	}}
}
