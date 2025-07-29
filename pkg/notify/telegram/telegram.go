package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
)

type Telegram struct {
	token  string
	chatId int64
	bot    *tgbotapi.BotAPI
}

func NewTelegram(token, chatId string) (*Telegram, error) {
	chatIdInt, _ := strconv.Atoi(chatId)
	service := &Telegram{
		token:  token,
		chatId: int64(chatIdInt),
	}
	bot, err := tgbotapi.NewBotAPI(service.token)
	if err != nil {
		return service, err
	}
	service.bot = bot
	return service, nil
}

func (t *Telegram) SendMsg(topMsg, msg string) {
	msg = topMsg + "\n" + msg
	botMsg := tgbotapi.NewMessage(t.chatId, msg)

	_, _ = t.bot.Send(botMsg)
}
