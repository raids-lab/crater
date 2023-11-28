package models

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
)

func JSONStringToList(str string) []string {
	list := []string{}
	if str == "" {
		return list
	}
	err := json.Unmarshal([]byte(str), &list)
	if err != nil {
		return []string{}
	}
	return list
}

func ListToJSONString(list []string) string {
	jsonBytes, err := json.Marshal(list)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

func JSONStringToMap(str string) map[string]string {
	m := map[string]string{}
	if str == "" {
		return m
	}
	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		return map[string]string{}
	}
	return m
}

func MapToJSONString(m map[string]string) string {
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

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
