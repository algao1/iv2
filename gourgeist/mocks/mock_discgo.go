package mocks

import (
	"fmt"
	"iv2/gourgeist/defs"
)

type Messager struct {
	Channels map[string][]defs.MessageData
}

func (m *Messager) SendMessage(msgData defs.MessageData, chName string) (uint64, error) {
	if _, ok := m.Channels[chName]; !ok {
		m.Channels[chName] = make([]defs.MessageData, 0)
	}
	m.Channels[chName] = append(m.Channels[chName], msgData)
	return 0, nil
}

func (m *Messager) GetMainMessage() (*defs.MessageData, error) {
	if msgs, ok := m.Channels["main"]; !ok || len(msgs) == 0 {
		return nil, fmt.Errorf("no message found")
	} else {
		return &msgs[len(msgs)-1], nil
	}
}

func (m *Messager) NewMainMessage(msgData defs.MessageData) error {
	m.SendMessage(msgData, "main")
	return nil
}

func (m *Messager) UpdateMainMessage(data defs.MessageData) error {
	// TODO: Finish implementation.
	return nil
}
