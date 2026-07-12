package botaction

import (
	"context"
	"encoding/json"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
)

func (d *Dispatcher) handleGetLoginInfo(ctx context.Context, principal models.BotPrincipal) (json.RawMessage, error) {
	bot, err := d.botRepo.FindByID(ctx, principal.BotID)
	if err != nil {
		return nil, onebot.NewError(onebot.RetCodePermissionDenied, "bot identity not found", nil)
	}
	return marshalData(loginInfoResponse{
		UserID:   principal.IdentityID.String(),
		Nickname: bot.Name,
	})
}

func (d *Dispatcher) handleGetStatus() (json.RawMessage, error) {
	return marshalData(statusResponse{
		Online: true,
		Good:   true,
	})
}

func (d *Dispatcher) handleGetVersionInfo() (json.RawMessage, error) {
	return marshalData(versionInfoResponse{
		Impl:          "PurrChat",
		Version:       onebot.ProfileVersion,
		OnebotVersion: "12",
	})
}
