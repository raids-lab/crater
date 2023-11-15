package models

import (
	"strings"
)

func FormatTaskAttrToModel(task *TaskAttr) *AITask {
	return &AITask{
		TaskName:        task.TaskName,
		UserName:        task.UserName,
		Namespace:       task.Namespace,
		TaskType:        task.TaskType,
		Image:           task.Image,
		ResourceRequest: ResourceListToJSON(task.ResourceRequest),
		WorkingDir:      task.WorkingDir,
		ShareDirs:       MapToJSONString(task.ShareDirs),
		Command:         task.Command,
		Args:            MapToJSONString(task.Args),
		SLO:             task.SLO,
		Status:          QueueingStatus,
	}
}

// TODO: directly return AITask
func FormatAITaskToAttr(model *AITask) *TaskAttr {
	resourceJson, _ := JSONToResourceList(model.ResourceRequest)
	return &TaskAttr{
		ID:              model.ID,
		TaskName:        model.TaskName,
		UserName:        model.UserName,
		Namespace:       model.Namespace,
		TaskType:        model.TaskType,
		Image:           model.Image,
		ResourceRequest: resourceJson,
		WorkingDir:      model.WorkingDir,
		ShareDirs:       JSONStringToMap(model.ShareDirs),
		Command:         model.Command,
		Args:            JSONStringToMap(model.Args),
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
