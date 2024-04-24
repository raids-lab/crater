package model

import "gorm.io/gorm"

// When creating a new project, a new data space will be created.
// A project can be associated with multiple spaces (belonging to other projects).
type Space struct {
	gorm.Model
	Path string `gorm:"uniqueIndex;type:varchar(256);not null;comment:数据空间根路径"`
	// space must belongs to one project
	ProjectID uint
	Project   Project
	// space can be associated with multiple projects in RW or RO mode
	ProjectSpaces []ProjectSpace
}
