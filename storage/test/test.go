package main

import (
	"fmt"
	"webdav/model"
	"webdav/orm"
)

func main() {
	db := orm.DB()
	var uuu []model.User
	db.Model(&model.User{}).Find(&uuu).Limit(1)
	fmt.Println(uuu[0].ID)
	fmt.Println(uuu[0].Name)
	fmt.Println(uuu[0].Model)
	fmt.Println(uuu[0].Nickname)
	for _, db_user := range uuu {
		if db_user.Role == model.RoleAdmin {
			fmt.Printf("role %d is admin\n", db_user.ID)
		} else if db_user.Role == model.RoleGuest {
			fmt.Printf("role %d is Guest\n", db_user.ID)
		} else if db_user.Role == model.RoleUser {
			fmt.Printf("role %d  is User\n", db_user.ID)
		}
	}

	// var bbb []model.Project
	// db.Model(&model.Project{}).First(&bbb)
	// fmt.Println(bbb[0].ID)
	// fmt.Println(bbb[0].Name)
	// fmt.Println(bbb[0].Status)
	// fmt.Println(bbb[0].ProjectSpaces)
	// service.CheckFilePermission("liyilong", "/files/project1/")

}
