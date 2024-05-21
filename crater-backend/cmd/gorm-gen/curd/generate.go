// Description: 生成所有表的 Model 结构体和 CRUD 代码
package main

import (
	"fmt"

	"github.com/raids-lab/crater/dao/model"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func ConnectPostgres() *gorm.DB {
	// Connect to the database
	dsn := `host=***REMOVED*** user=postgres password=***REMOVED*** 
		dbname=crater port=30432 sslmode=require TimeZone=Asia/Shanghai`
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect to postgres: %w", err))
	}
	return db
}

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "./dao/query",

		// gen.WithoutContext：禁用WithContext模式
		// gen.WithDefaultQuery：生成一个全局Query对象Q
		// gen.WithQueryInterface：生成Query接口
		Mode: gen.WithDefaultQuery | gen.WithQueryInterface,
	})

	// 通常复用项目中已有的SQL连接配置 db(*gorm.DB)
	g.UseDB(ConnectPostgres())

	// 从连接的数据库为所有表生成 Model 结构体和 CRUD 代码
	g.ApplyBasic(
		model.Project{},
		model.User{},
		model.UserProject{},
		model.Space{},
		model.ProjectSpace{},
		model.AIJob{},
		model.ImagePack{},
		model.ImageUpload{},
		model.Label{},
		model.Queue{},
		model.UserQueue{},
	)

	// 执行并生成代码
	g.Execute()
}
