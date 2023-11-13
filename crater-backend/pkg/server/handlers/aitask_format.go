package handlers

import (
	"strings"

	"github.com/aisystem/ai-protal/pkg/models"
)

func FormatTaskAttrToModel(task *models.TaskAttr) *models.AITask {
	return &models.AITask{
		TaskName:        task.TaskName,
		UserName:        task.UserName,
		Namespace:       task.Namespace,
		TaskType:        task.TaskType,
		Image:           task.Image,
		ResourceRequest: models.ResourceListToJSON(task.ResourceRequest),
		WorkingDir:      task.WorkingDir,
		ShareDirs:       strings.Join(task.ShareDirs, ","),
		Command:         task.Command,
		Args:            argsToString(task.Args),
		SLO:             task.SLO,
		Status:          models.QueueingStatus,
	}
}

// TODO: directly return models.AITask
func FormatAITaskToAttr(model *models.AITask) *models.TaskAttr {
	resourceJson, _ := models.JSONToResourceList(model.ResourceRequest)
	return &models.TaskAttr{
		ID:              model.ID,
		TaskName:        model.TaskName,
		UserName:        model.UserName,
		Namespace:       model.Namespace,
		TaskType:        model.TaskType,
		Image:           model.Image,
		ResourceRequest: resourceJson,
		WorkingDir:      model.WorkingDir,
		ShareDirs:       strings.Split(model.ShareDirs, ","),
		Command:         model.Command,
		Args:            dbstringToArgs(model.Args),
		SLO:             model.SLO,
		Status:          model.Status,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}
func argsToString(args map[string]string) string {
	str := ""
	var count = 0
	for key, value := range args {
		if count == len(args)-1 {
			str += key + "=" + value
		} else {
			str += key + "=" + value + " "
		}
		count++
	}
	return str
}

func dbstringToArgs(str string) map[string]string {
	args := map[string]string{}
	if len(str) == 0 {
		return args
	}
	ls := strings.Split(str, " ")
	for item := range ls {
		kv := strings.Split(ls[item], "=")
		args[kv[0]] = kv[1]
	}
	return args
}
