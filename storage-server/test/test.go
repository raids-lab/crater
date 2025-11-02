package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func main() {
	client := http.Client{}
	baseurl := "http://crater.act.buaa.edu.cn/api/ss/mydir"
	req, err := http.NewRequestWithContext(context.Background(), "POST", baseurl, http.NoBody)
	if err != nil {
		fmt.Println("can't create request")
		return
	}
	pathss := ""
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
	resp.Body.Close()
}
