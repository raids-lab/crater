package models

func FormatTaskAttrToModel(task *TaskAttr) *AITask {
	return &AITask{
		TaskName:        task.TaskName,
		UserName:        task.UserName,
		Namespace:       task.Namespace,
		TaskType:        task.TaskType,
		Image:           task.Image,
		ResourceRequest: ResourceListToJSON(task.ResourceRequest),
		WorkingDir:      task.WorkingDir,
		ShareDirs:       VolumesToJSONString(task.ShareDirs),
		Command:         task.Command,
		Args:            MapToJSONString(task.Args),
		SLO:             task.SLO,
		Status:          TaskQueueingStatus,
		EsitmatedTime:   task.EsitmatedTime,
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
		ShareDirs:       JSONStringToVolumes(model.ShareDirs),
		Command:         model.Command,
		Args:            JSONStringToMap(model.Args),
		SLO:             model.SLO,
		Status:          model.Status,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}
