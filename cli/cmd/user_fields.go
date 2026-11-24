package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

var userFieldsCmd = &cobra.Command{Use: "fields", Short: "Get enabled user fields and deployment labels", Args: noArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runRawRead(cmd, rawReadSpec{PayloadKey: "fields", Path: "/v1/system-config/user-fields", Params: noParams, Table: printRawObject})
	}}

func userFieldLabel(field map[string]interface{}, locale, key string) string {
	normalize := func(s string) string { return strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(s)) }
	labels := rawMap(field["labels"])
	keys := make([]string, 0, len(labels))
	for lang, value := range labels {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			keys = append(keys, lang)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := normalize(keys[i]), normalize(keys[j])
		if a == b {
			return keys[i] < keys[j]
		}
		return a < b
	})
	base := strings.Split(strings.ReplaceAll(locale, "_", "-"), "-")[0]
	if len(base) > 2 {
		base = base[:2]
	}
	for _, candidate := range []string{normalize(locale), normalize(base)} {
		for _, lang := range keys {
			if normalize(lang) == candidate {
				return labels[lang].(string)
			}
		}
	}
	if len(keys) > 0 {
		return labels[keys[0]].(string)
	}
	if key == "nickname" {
		return i18n.T("table_nickname")
	}
	names := map[string][2]string{
		"email": {"Email", "邮箱"}, "identity": {"Identity", "身份"}, "teacher": {"Advisor", "导师"},
		"group": {"Group", "组别"}, "phone": {"Phone", "手机号"}, "expiredAt": {"Expiration time", "过期时间"},
		"createdAt": {"Registration time", "注册时间"}, "lastLoginAt": {"Last login", "最后登录时间"},
	}
	if names, ok := names[key]; ok {
		if strings.HasPrefix(locale, "zh") {
			return names[1]
		}
		return names[0]
	}
	return key
}

func printUserFieldsTable(users []api.AdminUser, fields map[string]interface{}) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	headers := []string{i18n.T("table_id"), i18n.T("table_name"), i18n.T("table_role"), i18n.T("table_status")}
	for _, key := range keys {
		headers = append(headers, userFieldLabel(rawMap(fields[key]), i18n.GetCurrentLanguage(), key))
	}
	fmt.Println(strings.Join(headers, "\t"))
	for _, user := range users {
		row := []string{fmt.Sprint(user.ID), user.Name, userRoleLabel(user.Role), userStatusLabel(user.Status)}
		for _, key := range keys {
			value := user.Field(key)
			text := "—"
			if value != nil && value != "" {
				text = fmt.Sprint(value)
			}
			row = append(row, strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(text))
		}
		fmt.Println(strings.Join(row, "\t"))
	}
}
