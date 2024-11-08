package alert

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	config "github.com/raids-lab/crater/pkg/config"
)

type SMTPAlerter struct {
	smtpHost string
	smtpPort string
	smtpUser string
	smtpPass string
}

func newSMTPAlerter() alertHandlerInterface {
	smtpConfig := config.GetConfig()
	return &SMTPAlerter{
		smtpHost: smtpConfig.ACT.SMTP.Host,
		smtpPort: smtpConfig.ACT.SMTP.Port,
		smtpUser: smtpConfig.ACT.SMTP.User,
		smtpPass: smtpConfig.ACT.SMTP.Password,
	}
}

// 使用的服务器不支持tls，所以把smtpAuth相关的一些部分重写以绕过tls的检验。
type loginAuth struct {
	username, password string
}

func getLoginAuth(username, password string) smtp.Auth {
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

func (sa *SMTPAlerter) SendMessageTo(_ context.Context, receiver *model.UserAttribute, subject, body string) error {
	// 设置邮件发送者和接收者
	from := ***REMOVED***
	to := []string{*receiver.Email}

	// 设置邮件消息
	msg := []byte("To: " + to[0] + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body)

	conn, err := smtp.Dial(sa.smtpHost + ":" + sa.smtpPort)
	if err != nil {
		fmt.Println(err)
		return err
	}

	auth := getLoginAuth(sa.smtpUser, sa.smtpPass)
	if err = conn.Auth(auth); err != nil {
		fmt.Println("autherr:", err)
		return err
	}

	err = smtp.SendMail(sa.smtpHost+":"+sa.smtpPort, auth, from, to, msg)

	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("Email sent!")
	return nil
}
