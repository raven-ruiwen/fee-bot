package notify

import (
	"fee-bot/pkg/notify/slack"
	"fee-bot/pkg/notify/telegram"
	"fmt"
)

const (
	PlatformSlack    = "slack"
	PlatformTelegram = "telegram"
)

type INotify interface {
	SendMsg(topMsg, msg string)
}

type Service struct {
	notifies []INotify
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) AddNotify(platform, token, channel string) {
	switch platform {
	case PlatformSlack:
		service := slack.NewSlack(token, channel)
		s.notifies = append(s.notifies, service)
	case PlatformTelegram:
		service, err := telegram.NewTelegram(token, channel)
		if err != nil {
			fmt.Printf("add notify telegram failed: %s", err.Error())
			return
		}
		s.notifies = append(s.notifies, service)
	}
}

func (s *Service) SendMsg(topMsg, msg string) {
	for _, notify := range s.notifies {
		go notify.SendMsg(topMsg, msg)
	}
}
