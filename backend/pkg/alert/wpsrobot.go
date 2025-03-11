package alert

// TODO(zengqile25): WPS Robot 的具体实现

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"net/http"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
)

// WebhookAddress 是机器人Webhook地址
var WebhookAddress = config.GetConfig().WPSRobot.WebhookAddress

// Message 是发送的消息结构体
type Message struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}
type wpsRobot struct{}

func newWPSRobot() alertHandlerInterface {
	return &wpsRobot{}
}

// SendMessage 发送文本消息到WPS群聊
func (w *wpsRobot) SendMessageTo(ctx context.Context, _ *model.UserAttribute, _, body string) error {
	msg := Message{
		Msgtype: "text",
		Text: struct {
			Content string `json:"content"`
		}{
			Content: body,
		},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", WebhookAddress, bytes.NewBuffer(msgBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}
func TestSendMessageTo() {
	ctx := context.Background()
	robot := newWPSRobot()

	receiver := &model.UserAttribute{}
	subject := "Test Subject"
	body := "This is a test message."

	err := robot.SendMessageTo(ctx, receiver, subject, body)
	if err == nil {
		log.Println("SendMessageTo succeeded")
	}
}
