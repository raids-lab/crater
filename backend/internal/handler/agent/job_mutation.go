package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
)

type agentJobNameArgs struct {
	JobName string `json:"job_name"`
}

func isAgentOwnedJobMutationTool(toolName string) bool {
	switch toolName {
	case agentToolDeleteJob, agentToolStopJob, agentToolResubmitJob:
		return true
	default:
		return false
	}
}

func agentOwnedJobMutationActionName(toolName string) string {
	switch toolName {
	case agentToolDeleteJob:
		return "删除"
	case agentToolStopJob:
		return "停止"
	case agentToolResubmitJob:
		return "重提"
	default:
		return "操作"
	}
}

func (mgr *AgentMgr) validateOwnedJobMutationBeforeConfirmation(
	c *gin.Context,
	token util.JWTMessage,
	toolName string,
	rawArgs json.RawMessage,
) error {
	if !isAgentOwnedJobMutationTool(toolName) {
		return nil
	}

	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "invalid args")
	}
	args.JobName = strings.TrimSpace(args.JobName)
	if args.JobName == "" {
		return bizerr.BadRequest.MissingParameter.New("job_name is required")
	}

	j := query.Job
	jobQuery := j.WithContext(c).Where(j.JobName.Eq(args.JobName))
	if token.RolePlatform != model.RoleAdmin {
		jobQuery = jobQuery.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	if _, err := jobQuery.First(); err != nil {
		return bizerr.Forbidden.PermissionDenied.New(
			fmt.Sprintf("该作业不存在或你没有访问权限，不能发起%s确认", agentOwnedJobMutationActionName(toolName)),
		)
	}
	return nil
}

func mergeToolArgsWithPayload(baseArgs, payload json.RawMessage) (json.RawMessage, error) {
	if len(payload) == 0 || string(payload) == "null" {
		return baseArgs, nil
	}

	base := make(map[string]any)
	if len(baseArgs) > 0 {
		if err := json.Unmarshal(baseArgs, &base); err != nil {
			return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid stored tool args")
		}
	}

	incoming := make(map[string]any)
	if err := json.Unmarshal(payload, &incoming); err != nil {
		return nil, bizerr.BadRequest.ParameterError.Wrap(err, "invalid confirmation payload")
	}

	for key, value := range incoming {
		base[key] = value
	}

	merged, err := json.Marshal(base)
	if err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(err, "failed to merge confirmation payload")
	}
	return merged, nil
}
