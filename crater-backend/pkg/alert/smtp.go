package alert

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	config "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
)

type SMTPAlerter struct {
	smtpHost string
	smtpPort string

	auth smtp.Auth
}

func newSMTPAlerter() (alertHandlerInterface, error) {
	smtpConfig := config.GetConfig()
	smtpHost := smtpConfig.ACT.SMTP.Host
	smtpPort := smtpConfig.ACT.SMTP.Port

	conn, err := smtp.Dial(smtpHost + ":" + smtpPort)
	if err != nil {
		return nil, err
	}

	auth := getLoginAuth(smtpConfig.ACT.SMTP.User, smtpConfig.ACT.SMTP.Password)
	if err := conn.Auth(auth); err != nil {
		return nil, err
	}
	return &SMTPAlerter{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		auth:     auth,
	}, nil
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
	if receiver.Email == nil {
		logutils.Log.Warnf("%s does not have an email address", receiver.Name)
		return nil
	}

	// 设置邮件发送者和接收者
	from := ***REMOVED***
	to := []string{*receiver.Email}

	// 设置邮件消息
	msg := []byte("To: " + to[0] + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body)

	if err := smtp.SendMail(sa.smtpHost+":"+sa.smtpPort, sa.auth, from, to, msg); err != nil {
		logutils.Log.Errorf("Failed to send email to %s: %v", to[0], err)
		return err
	}

	logutils.Log.Infof("Sent email to %s", to[0])
	return nil
}
