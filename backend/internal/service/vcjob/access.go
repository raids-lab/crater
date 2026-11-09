package vcjob

import (
	"context"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
)

// VisibleJobByName scopes a job lookup to the platform admin or the owner in the active account.
func VisibleJobByName(ctx context.Context, name string, token util.JWTMessage) query.IJobDo {
	j := query.Job
	jobs := j.WithContext(ctx).Where(j.JobName.Eq(name))
	if token.RolePlatform != model.RoleAdmin {
		jobs = jobs.Where(j.AccountID.Eq(token.AccountID), j.UserID.Eq(token.UserID))
	}
	return jobs
}
