package mysmtp

import (
	"fmt"
	"net/smtp"
	"strings"

	config "github.com/raids-lab/crater/pkg/config"
)

// 使用的服务器不支持tls，所以把smtpAuth相关的一些部分重写以绕过tls的检验。
type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (proto string, toServe []byte, err error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	command := string(fromServer)
	command = strings.TrimSpace(command)
	command = strings.TrimSuffix(command, ":")
	command = strings.ToLower(command)

	if more {
		if command == "username" {
			return []byte(a.username), nil
		} else if command == "password" {
			return []byte(a.password), nil
		} else {
			// We've already sent everything.
			return nil, fmt.Errorf("unexpected server challenge: %s", command)
		}
	}
	return nil, nil
}

// 可以通过下面例子调用函数发送邮件
//
//	 err := mysmtp.SendEmail("邮箱", "主体", "内容")
//		if err != nil {
//			fmt.Println(err)
//		}
func SendEmail(receiver, subject, body string) error {
	smtpConfig := config.GetConfig()
	smtpHost := smtpConfig.ACT.SMTP.Host
	smtpPort := smtpConfig.ACT.SMTP.Port
	smtpUser := smtpConfig.ACT.SMTP.User
	smtpPass := smtpConfig.ACT.SMTP.Password

	// 设置邮件发送者和接收者
	from := ***REMOVED***
	to := []string{receiver}

	// 设置邮件消息
	msg := []byte("To: " + to[0] + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body)

	conn, err := smtp.Dial(smtpHost + ":" + smtpPort)
	if err != nil {
		fmt.Println(err)
		return err
	}

	auth := LoginAuth(smtpUser, smtpPass)
	if err = conn.Auth(auth); err != nil {
		fmt.Println("autherr:", err)
		return err
	}

	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)

	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("Email sent!")
	return nil
}
