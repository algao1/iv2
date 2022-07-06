package mocks

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

type Messager struct {
	Channels map[string][]discord.Message
}

func (m *Messager) SendMessage(msgData api.SendMessageData, chName string) (discord.MessageID, error) {
	if _, ok := m.Channels[chName]; !ok {
		m.Channels[chName] = make([]discord.Message, 0)
	}
	m.Channels[chName] = append(m.Channels[chName],
		discord.Message{
			Content: msgData.Content,
			Embeds:  msgData.Embeds,
		},
	)
	return 0, nil
}

func (m *Messager) GetMainMessage() (*discord.Message, error) {
	if msgs, ok := m.Channels["main"]; !ok || len(msgs) == 0 {
		return nil, fmt.Errorf("no message found")
	} else {
		return &msgs[len(msgs)-1], nil
	}
}

func (m *Messager) NewMainMessage(msgData api.SendMessageData) error {
	m.SendMessage(msgData, "main")
	return nil
}

func (m *Messager) UpdateMainMessage(data api.EditMessageData) error {
	// TODO: Finish implementation.
	return nil
}
