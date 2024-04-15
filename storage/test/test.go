package main

import (
	"context"
	"fmt"
	"net/http"
)

func main() {

	client := http.Client{}
	baseurl := "http://localhost:8086/v1/storage/verify"
	req, err := http.NewRequestWithContext(context.Background(), "GET", baseurl, http.NoBody)
	if err != nil {
		fmt.Println("can't create request")
		return
	}
	req.Header.Set("Authorization", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjgsInBpZCI6OSwicHJvIjozLCJjaWQiOjEsImNybyI6MiwicGxmIjozLCJleHAiOjE3MTI5MTQwOTh9.YmkatSfYnfvaLOjiBExtqsMft-kf0MGoXFdjn5mHv8M")
	req.Header.Set("Content-Type", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("body:", resp.Body)
		return
	}
	defer resp.Body.Close()

	// var bbb []model.Project
	// db.Model(&model.Project{}).First(&bbb)
	// fmt.Println(bbb[0].ID)
	// fmt.Println(bbb[0].Name)
	// fmt.Println(bbb[0].Status)
	// fmt.Println(bbb[0].ProjectSpaces)
	// service.CheckFilePermission("liyilong", "/files/project1/")

}
