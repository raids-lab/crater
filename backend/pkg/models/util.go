package models

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
)

func JSONToResourceList(str string) (v1.ResourceList, error) {
	res := v1.ResourceList{}
	if str == "" {
		return res, nil
	}
	err := json.Unmarshal([]byte(str), &res)
	if err != nil {
		return nil, err
	}
	return res, nil

}

func ResourceListToJSON(rs v1.ResourceList) string {
	jsonBytes, err := json.Marshal(rs)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}
