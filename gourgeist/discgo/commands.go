package discgo

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"go.uber.org/zap"
)

func registeredCommands() []api.CreateCommandData {
	commands := []api.CreateCommandData{}
	return commands
}

func InteractionCreateHandler(ses *session.Session, logger *zap.Logger) func(*gateway.InteractionCreateEvent) {
	return func(e *gateway.InteractionCreateEvent) {
		var resp api.InteractionResponse
		var err error

		switch data := e.Data.(type) {
		case *discord.CommandInteraction:
			resp, err = handleCommand(data, logger)
		}

		if err != nil {
			return
		}

		if err := ses.RespondInteraction(e.ID, e.Token, resp); err != nil {
			return
		}
	}
}

func handleCommand(data *discord.CommandInteraction, logger *zap.Logger) (api.InteractionResponse, error) {
	var resp api.InteractionResponse
	switch data.Name {
	default:
		return resp, fmt.Errorf("unknown command: %s", data.Name)
	}
}
