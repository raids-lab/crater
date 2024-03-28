// Description: 生成所有表的 Model 结构体和 CRUD 代码
package main

import (
	"fmt"

	"github.com/raids-lab/crater/pkg/model"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

const MySQLDSN = "root:buaak8sportal@2023mysql@tcp(***REMOVED***:30306)/crater?charset=utf8mb4&parseTime=True"

func ConnectDB(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(fmt.Errorf("connect db fail: %w", err))
	}
	return db
}

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "../../pkg/query",

		// gen.WithoutContext：禁用WithContext模式
		// gen.WithDefaultQuery：生成一个全局Query对象Q
		// gen.WithQueryInterface：生成Query接口
		Mode: gen.WithDefaultQuery | gen.WithQueryInterface,
	})

	// 通常复用项目中已有的SQL连接配置 db(*gorm.DB)
	g.UseDB(ConnectDB(MySQLDSN))

	// 从连接的数据库为所有表生成 Model 结构体和 CRUD 代码
	g.ApplyBasic(model.Project{}, model.User{}, model.UserProject{})

	// 执行并生成代码
	g.Execute()
}
