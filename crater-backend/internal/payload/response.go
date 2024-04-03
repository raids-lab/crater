package payload

import "github.com/raids-lab/crater/dao/model"

// 定义返回值时，优先在使用到该返回值的 /internal/handler/xxx.go 中直接定义
// 当某个返回值的结构体通用时，从 /internal/handler/xxx.go 中提升至此文件中

type ProjectResp struct {
	ID         uint         `json:"id"`         // 项目ID
	Name       string       `json:"name"`       // 项目名称
	Role       model.Role   `json:"role"`       // 用户在项目中的角色
	IsPersonal bool         `json:"isPersonal"` // 是否为个人项目
	Status     model.Status `json:"status"`     // 项目状态
}
