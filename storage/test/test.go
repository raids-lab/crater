package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	client := http.Client{}
	baseurl := "http://crater.act.buaa.edu.cn/api/ss/mydir"
	req, err := http.NewRequest("POST", baseurl, http.NoBody)
	if err != nil {
		fmt.Println("can't create request")
		return
	}
	pathss := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjgsInBpZCI6OSwicHJvIjozLCJjaWQiOjEsImNybyI6MiwicGxmIjozLCJleHAiOjE3MTMyNjk3Mzl9.nkA0yOVh1IdZ1ek2NMelCAQ_3764_YwXQ6Ik4e-LXr4"
	req.Header.Set("Authorization", "Bearer "+pathss)

	req.Header.Set("accept", "application/json")
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
	// var tokenResp TokenResp
	// if err := json.Unmarshal([]byte(string(body)), &tokenResp); err != nil {
	// 	fmt.Println("541223")
	// }
	// fmt.Println(tokenResp.Code, tokenResp.Data.UserId)
	defer resp.Body.Close()
}
