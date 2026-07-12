package botaction

import (
	"context"
	"encoding/json"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"

	"github.com/google/uuid"
)

func (d *Dispatcher) handleGetConversationInfo(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	params, err := onebot.DecodeParams[conversationIDParams](request)
	if err != nil {
		return nil, err
	}

	conversationID, err := parseConversationID(params.ConversationID)
	if err != nil {
		return nil, err
	}

	if err := d.authorize(ctx, principal, conversationID, ""); err != nil {
		return nil, err
	}

	conv, err := d.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, onebot.NewError(onebot.RetCodeResourceNotFound, "conversation not found", nil)
	}

	return marshalData(conversationInfoResponse{
		ConversationID:   conv.ID.String(),
		ConversationType: string(conv.ConversationType),
		Name:             conv.Name,
		AvatarURL:        conv.AvatarURL,
	})
}

func (d *Dispatcher) handleGetConversationList(ctx context.Context, principal models.BotPrincipal) (json.RawMessage, error) {
	installations, err := d.installationRepo.FindByApp(ctx, principal.BotID)
	if err != nil {
		return nil, onebot.NewError(onebot.RetCodeInternal, "failed to list installations", err)
	}

	result := make([]conversationInfoResponse, 0, len(installations))
	for _, inst := range installations {
		if inst.Status != models.InstallationActive {
			continue
		}

		var conv *models.Conversation
		if inst.TargetType == models.InstallationTargetConversation {
			conv, err = d.conversationRepo.FindByID(ctx, inst.TargetID)
		} else {
			conv, err = d.conversationRepo.FindByUsers(ctx, principal.BotID, inst.TargetID)
		}
		if err != nil || conv == nil {
			continue
		}

		result = append(result, conversationInfoResponse{
			ConversationID:   conv.ID.String(),
			ConversationType: string(conv.ConversationType),
			Name:             conv.Name,
			AvatarURL:        conv.AvatarURL,
		})
	}

	return marshalData(result)
}

func (d *Dispatcher) handleGetMemberList(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	params, err := onebot.DecodeParams[conversationIDParams](request)
	if err != nil {
		return nil, err
	}

	conversationID, err := parseConversationID(params.ConversationID)
	if err != nil {
		return nil, err
	}

	if err := d.authorize(ctx, principal, conversationID, models.CapabilityMembersRead); err != nil {
		return nil, err
	}

	enrollments, err := d.enrollmentRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return nil, onebot.NewError(onebot.RetCodeInternal, "failed to list members", err)
	}

	result := make([]memberInfoResponse, 0, len(enrollments))
	for _, enr := range enrollments {
		user, err := d.userRepo.FindByID(ctx, enr.UserID)
		if err != nil || user == nil {
			continue
		}
		result = append(result, memberInfoResponse{
			UserID:   user.ID.String(),
			Nickname: user.Username,
			Role:     string(enr.Role),
			IsBot:    user.IsBot,
		})
	}

	return marshalData(result)
}

func (d *Dispatcher) handleGetMemberInfo(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	params, err := onebot.DecodeParams[memberInfoParams](request)
	if err != nil {
		return nil, err
	}

	conversationID, err := parseConversationID(params.ConversationID)
	if err != nil {
		return nil, err
	}

	if err := onebot.ValidateOpaqueID("user_id", params.UserID); err != nil {
		return nil, err
	}
	targetUserID, err := uuid.Parse(params.UserID)
	if err != nil {
		return nil, onebot.NewError(onebot.RetCodeInvalidParams, "invalid user_id", err)
	}

	if err := d.authorize(ctx, principal, conversationID, models.CapabilityMembersRead); err != nil {
		return nil, err
	}

	enrollment, err := d.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, targetUserID)
	if err != nil {
		return nil, onebot.NewError(onebot.RetCodeResourceNotFound, "member not found in this conversation", nil)
	}

	user, err := d.userRepo.FindByID(ctx, targetUserID)
	if err != nil || user == nil {
		return nil, onebot.NewError(onebot.RetCodeResourceNotFound, "member not found", nil)
	}

	return marshalData(memberInfoResponse{
		UserID:   user.ID.String(),
		Nickname: user.Username,
		Role:     string(enrollment.Role),
		IsBot:    user.IsBot,
	})
}
