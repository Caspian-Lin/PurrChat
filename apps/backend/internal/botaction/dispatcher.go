package botaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"purr-chat-server/internal/messaging"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"

	"github.com/google/uuid"
)

// Dispatcher 实现 botws.ActionDispatcher，供 Universal WS 和 HTTP 共用。
// 身份来源为 models.BotPrincipal（由 Bot credential middleware 验证），
// 绝不从 request params 读取 bot_id/user_id 作为身份。
type Dispatcher struct {
	messageSender    messaging.BotMessageSender
	botRepo          repository.BotRepository
	userRepo         repository.UserRepository
	conversationRepo repository.ConversationRepository
	enrollmentRepo   repository.EnrollmentRepository
	messageRepo      repository.ConversationMessageRepository
	installationRepo repository.BotInstallationRepository
}

func NewDispatcher(
	messageSender messaging.BotMessageSender,
	botRepo repository.BotRepository,
	userRepo repository.UserRepository,
	conversationRepo repository.ConversationRepository,
	enrollmentRepo repository.EnrollmentRepository,
	messageRepo repository.ConversationMessageRepository,
	installationRepo repository.BotInstallationRepository,
) *Dispatcher {
	return &Dispatcher{
		messageSender:    messageSender,
		botRepo:          botRepo,
		userRepo:         userRepo,
		conversationRepo: conversationRepo,
		enrollmentRepo:   enrollmentRepo,
		messageRepo:      messageRepo,
		installationRepo: installationRepo,
	}
}

// Dispatch 路由 OneBot Action 请求到对应 handler。
func (d *Dispatcher) Dispatch(ctx context.Context, principal models.BotPrincipal, request onebot.ActionRequest) (json.RawMessage, error) {
	definition, err := onebot.ResolveAction(request.Action)
	if err != nil {
		return nil, err
	}

	switch definition.Name {
	case "send_message":
		return d.handleSendMessage(ctx, principal, request)
	case "get_login_info":
		return d.handleGetLoginInfo(ctx, principal)
	case "get_status":
		return d.handleGetStatus()
	case "get_version_info":
		return d.handleGetVersionInfo()
	case "get_conversation_info":
		return d.handleGetConversationInfo(ctx, principal, request)
	case "get_conversation_list":
		return d.handleGetConversationList(ctx, principal)
	case "get_conversation_member_list":
		return d.handleGetMemberList(ctx, principal, request)
	case "get_conversation_member_info":
		return d.handleGetMemberInfo(ctx, principal, request)
	case "get_message_history":
		return d.handleGetMessageHistory(ctx, principal, request)
	default:
		return nil, onebot.NewError(onebot.RetCodeUnsupportedAction, "action is not implemented: "+definition.Name, nil)
	}
}

// authorize 校验 Bot 对目标会话的读取权限。
// 验证：Bot active → 会话成员 → active installation（direct 查 user target，group 查 conversation target）→ capability。
// capability 为空时跳过 capability 检查，但仍执行其他校验。
func (d *Dispatcher) authorize(ctx context.Context, principal models.BotPrincipal, conversationID uuid.UUID, capability string) error {
	bot, err := d.botRepo.FindByID(ctx, principal.BotID)
	if err != nil || bot.Status != models.BotStatusActive {
		return onebot.NewError(onebot.RetCodePermissionDenied, "bot is not available", nil)
	}

	if _, err := d.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, principal.BotID); err != nil {
		return onebot.NewError(onebot.RetCodePermissionDenied, "bot is not a member of this conversation", nil)
	}

	conv, err := d.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return onebot.NewError(onebot.RetCodeResourceNotFound, "conversation not found", nil)
	}

	members, err := d.enrollmentRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return onebot.NewError(onebot.RetCodeInternal, "failed to verify conversation members", err)
	}

	var inst *models.BotInstallation
	if conv.ConversationType == models.ConversationTypeDirect {
		if len(members) != 2 {
			return onebot.NewError(onebot.RetCodePermissionDenied, "invalid direct conversation", nil)
		}
		var targetUserID uuid.UUID
		for _, m := range members {
			if m.UserID != principal.BotID {
				targetUserID = m.UserID
			}
		}
		if targetUserID == uuid.Nil {
			return onebot.NewError(onebot.RetCodePermissionDenied, "invalid direct conversation", nil)
		}
		inst, err = d.installationRepo.FindByAppAndTarget(ctx, principal.BotID, models.InstallationTargetUser, targetUserID)
	} else {
		inst, err = d.installationRepo.FindByAppAndTarget(ctx, principal.BotID, models.InstallationTargetConversation, conversationID)
	}

	if err != nil || inst == nil {
		return onebot.NewError(onebot.RetCodeInstallationInactive, "no active installation found", nil)
	}
	if inst.Status != models.InstallationActive {
		return onebot.NewError(onebot.RetCodeInstallationInactive, "bot installation is not active", nil)
	}
	if capability != "" && !models.HasCapability(inst.GrantedCapabilities, capability) {
		return onebot.NewError(onebot.RetCodeCapabilityRequired, "required capability not granted: "+capability, nil)
	}
	return nil
}

// parseConversationID 解析并校验 conversation_id 参数。
func parseConversationID(raw string) (uuid.UUID, error) {
	if err := onebot.ValidateOpaqueID("conversation_id", raw); err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, onebot.NewError(onebot.RetCodeInvalidParams, fmt.Sprintf("invalid conversation_id: %s", raw), err)
	}
	return id, nil
}

// marshalData 将 response struct 序列化为 json.RawMessage。
func marshalData(v any) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.New("failed to encode response")
	}
	return data, nil
}
