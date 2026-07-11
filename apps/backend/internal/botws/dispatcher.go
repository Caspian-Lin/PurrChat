package botws

import (
	"context"
	"encoding/json"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
)

type ActionDispatcher interface {
	Dispatch(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error)
}

type RegistryDispatcher struct{}

func (RegistryDispatcher) Dispatch(_ context.Context, _ models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	definition, err := onebot.ResolveAction(request.Action)
	if err != nil {
		return nil, err
	}
	return nil, onebot.NewError(onebot.RetCodeUnsupportedAction, "action is not implemented: "+definition.Name, nil)
}
