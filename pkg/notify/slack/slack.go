package slack

import (
	"encoding/json"
	"fmt"
	"github.com/imroc/req"
)

type Slack struct {
	token   string
	channel string
}

func NewSlack(token, channel string) *Slack {
	return &Slack{
		token:   token,
		channel: channel,
	}
}

func (t *Slack) SendMsg(topMsg string, msg string) {
	header := req.Header{}
	header["Authorization"] = fmt.Sprintf("Bearer %s", t.token)
	header["Content-Type"] = "application/json"

	b := SlackBlock{}
	b.Type = "section"
	b.Text.Type = "mrkdwn"
	b.Text.Text = msg

	sMsg := SlackMsg{
		Channel: t.channel,
		Text:    topMsg,
	}
	sMsg.Blocks = append(sMsg.Blocks, b)

	data, _ := json.Marshal(sMsg)

	_, err := req.Post("https://slack.com/api/chat.postMessage", header, string(data))
	if err != nil {
		fmt.Println(err.Error())
	}
}

type SlackMsg struct {
	Channel string       `json:"channel"`
	Text    string       `json:"text"`
	Blocks  []SlackBlock `json:"blocks"`
}

type SlackBlock struct {
	Type string `json:"type"`
	Text struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"text"`
}
