package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// 业务错误(handler 通过 containsBadInput 判断返回 400)
var (
	errBotNotFound          = errors.New("bot not found")
	errBotDisabled          = errors.New("bot is disabled")
	errInstallNotMember     = errors.New("not a conversation member")
	errInstallNoPermission  = errors.New("only conversation owner/admin can install bots")
	errCannotInstallOther   = errors.New("cannot install a bot for another user")
	errBotNotDiscoverable   = errors.New("this bot is not publicly available")
	errAlreadyInstalled     = errors.New("bot is already installed to this target")
	errInstallationNotFound = errors.New("installation not found")
	errGrantedExceedsReq    = errors.New("granted capabilities exceed requested capabilities")
)

// InstallationService Bot 安装业务逻辑服务
type InstallationService struct {
	installationRepo repository.BotInstallationRepository
	botRepo          repository.BotRepository
	enrollmentRepo   repository.EnrollmentRepository
	messageRepo      repository.ConversationMessageRepository
}

// NewInstallationService 创建安装服务
func NewInstallationService(
	installationRepo repository.BotInstallationRepository,
	botRepo repository.BotRepository,
	enrollmentRepo repository.EnrollmentRepository,
	messageRepo repository.ConversationMessageRepository,
) *InstallationService {
	return &InstallationService{
		installationRepo: installationRepo,
		botRepo:          botRepo,
		enrollmentRepo:   enrollmentRepo,
		messageRepo:      messageRepo,
	}
}

// CreateInstallation 安装 Bot 到用户私聊或群聊会话
func (s *InstallationService) CreateInstallation(ctx context.Context, installerID string, appID string, req *models.CreateInstallationRequest) (*models.BotInstallation, error) {
	installerUUID, err := uuid.Parse(installerID)
	if err != nil {
		return nil, err
	}
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}

	// 1. 获取 Bot 并校验状态
	bot, err := s.botRepo.FindByID(ctx, appUUID)
	if err != nil {
		return nil, errBotNotFound
	}
	if bot.Status != models.BotStatusActive {
		return nil, errBotDisabled
	}

	// 2. 权限校验
	switch req.TargetType {
	case models.InstallationTargetUser:
		// 用户只能给自己安装
		if req.TargetID != installerUUID {
			return nil, errCannotInstallOther
		}
		// 非 owner 只能安装可发现的 Bot(listed/featured)
		if bot.OwnerID != installerUUID && bot.Discoverability == models.DiscoverabilityUnlisted {
			return nil, errBotNotDiscoverable
		}
	case models.InstallationTargetConversation:
		// 群聊安装:操作者必须是 conversation owner/admin
		enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, req.TargetID, installerUUID)
		if err != nil || enrollment == nil {
			return nil, errInstallNotMember
		}
		if enrollment.Role != models.EnrollmentRoleOwner && enrollment.Role != models.EnrollmentRoleAdmin {
			return nil, errInstallNoPermission
		}
	}

	// 3. 幂等:已安装则返回现有记录
	if existing, _ := s.installationRepo.FindByAppAndTarget(ctx, appUUID, req.TargetType, req.TargetID); existing != nil {
		return nil, errAlreadyInstalled
	}

	// 4. granted_capabilities:未指定则授予 Bot 声明的全部 requested
	granted := req.GrantedCapabilities
	if len(granted) == 0 {
		granted = bot.RequestedCapabilities
	}
	// 校验 granted ⊆ requested(安装者只能缩减,不能超授权)
	if violations := models.IsGrantedSubsetOfRequested(granted, bot.RequestedCapabilities); len(violations) > 0 {
		return nil, errGrantedExceedsReq
	}

	// 5. diagnostics_consent:外发 Bot 强制 granted(数据已必然到达 owner 经第三方)
	diag := req.DiagnosticsConsent
	if diag == "" {
		diag = models.DiagnosticsDenied
	}
	if models.HasCapability(bot.RequestedCapabilities, models.CapabilityNetworkExternal) {
		diag = models.DiagnosticsGranted
	}

	// 6. 群聊安装:Bot 作为 enrollment(member)加入会话(消息路由仍依赖 enrollment)
	if req.TargetType == models.InstallationTargetConversation {
		botEnrollment := &models.Enrollment{
			ConversationID: req.TargetID,
			UserID:         appUUID,
			Role:           models.EnrollmentRoleMember,
			JoinedAt:       time.Now().UTC(),
		}
		if err := s.enrollmentRepo.Create(ctx, botEnrollment); err != nil {
			logger.ErrorfWithCaller("[InstallationService] Failed to enroll bot in conversation: %v", err)
		}

		// 声明 network:external 的 Bot 安装到群聊后,强制向全体成员发系统消息告知外发
		if models.HasCapability(bot.RequestedCapabilities, models.CapabilityNetworkExternal) {
			s.notifyExternalBotInstalled(ctx, bot, req.TargetID, installerUUID)
		}
	}

	// 7. 创建安装记录
	installation := &models.BotInstallation{
		AppID:               appUUID,
		InstalledBy:         installerUUID,
		TargetType:          req.TargetType,
		TargetID:            req.TargetID,
		GrantedCapabilities: granted,
		DiagnosticsConsent:  diag,
		Status:              models.InstallationActive,
	}
	if err := s.installationRepo.Create(ctx, installation); err != nil {
		return nil, err
	}
	return installation, nil
}

// GetInstallation 获取单个安装详情(带 Bot 关联)
func (s *InstallationService) GetInstallation(ctx context.Context, requesterID, installationID string) (*models.BotInstallation, error) {
	requesterUUID, err := parseID(requesterID)
	if err != nil {
		return nil, err
	}
	id, err := parseID(installationID)
	if err != nil {
		return nil, err
	}
	inst, err := s.installationRepo.FindByIDWithApp(ctx, id)
	if err != nil {
		return nil, ErrResourceNotFound
	}
	if !s.canManage(ctx, inst, requesterUUID) {
		return nil, ErrResourceNotFound
	}
	return inst, nil
}

// AuthorizeBotConversationRead 校验可信 Bot 身份对会话读取能力的实时授权。
// 调用方必须先通过 Bot credential 验证 botID，不能接受客户端伪造的身份。
func (s *InstallationService) AuthorizeBotConversationRead(ctx context.Context, botID, conversationID uuid.UUID, capability string) error {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil || bot.Status != models.BotStatusActive {
		return ErrResourceNotFound
	}
	if err := requireConversationMember(ctx, s.enrollmentRepo, conversationID, botID); err != nil {
		return ErrResourceNotFound
	}
	inst, err := s.installationRepo.FindByAppAndTarget(ctx, botID, models.InstallationTargetConversation, conversationID)
	if err != nil || inst.Status != models.InstallationActive || !models.HasCapability(inst.GrantedCapabilities, capability) {
		return ErrResourceNotFound
	}
	return nil
}

// ListByApp 列出某 Bot 的所有安装(仅 Bot owner)
func (s *InstallationService) ListByApp(ctx context.Context, ownerID string, appID string) ([]*models.BotInstallation, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, err
	}
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	bot, err := s.botRepo.FindByID(ctx, appUUID)
	if err != nil {
		return nil, errBotNotFound
	}
	if bot.OwnerID != ownerUUID {
		return nil, errNotAuthorized
	}
	return s.installationRepo.FindByApp(ctx, appUUID)
}

// ListByTarget 列出某目标的安装(用户自己的私聊 或 会话的群聊)
func (s *InstallationService) ListByTarget(ctx context.Context, requesterID string, targetType models.InstallationTargetType, targetID string) ([]*models.BotInstallation, error) {
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, err
	}
	targetUUID, err := uuid.Parse(targetID)
	if err != nil {
		return nil, err
	}

	switch targetType {
	case models.InstallationTargetUser:
		// 只能查自己的私聊安装
		if targetUUID != requesterUUID {
			return nil, errNotAuthorized
		}
	case models.InstallationTargetConversation:
		// 必须是会话成员
		enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, targetUUID, requesterUUID)
		if err != nil || enrollment == nil {
			return nil, errInstallNotMember
		}
	}

	return s.installationRepo.FindByTarget(ctx, targetType, targetUUID)
}

// ListMine 列出当前用户作为安装者的安装
func (s *InstallationService) ListMine(ctx context.Context, installerID string) ([]*models.BotInstallation, error) {
	installerUUID, err := uuid.Parse(installerID)
	if err != nil {
		return nil, err
	}
	return s.installationRepo.FindByInstaller(ctx, installerUUID)
}

// UpdateInstallation 更新安装(暂停/恢复/重新授权;权限:installer 或会话 admin/owner 或 Bot owner)
func (s *InstallationService) UpdateInstallation(ctx context.Context, requesterID string, installationID string, req *models.UpdateInstallationRequest) (*models.BotInstallation, error) {
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(installationID)
	if err != nil {
		return nil, err
	}

	inst, err := s.installationRepo.FindByIDWithApp(ctx, id)
	if err != nil {
		return nil, errInstallationNotFound
	}

	if !s.canManage(ctx, inst, requesterUUID) {
		return nil, errNotAuthorized
	}

	// 外发 Bot 的 diagnostics 不可降级为 denied
	if req.DiagnosticsConsent == models.DiagnosticsDenied && inst.App != nil &&
		models.HasCapability(inst.App.RequestedCapabilities, models.CapabilityNetworkExternal) {
		req.DiagnosticsConsent = models.DiagnosticsGranted
	}

	if req.Status != "" {
		inst.Status = req.Status
	}
	if req.GrantedCapabilities != nil {
		inst.GrantedCapabilities = req.GrantedCapabilities
	}
	if req.DiagnosticsConsent != "" {
		inst.DiagnosticsConsent = req.DiagnosticsConsent
	}

	if err := s.installationRepo.Update(ctx, inst); err != nil {
		return nil, err
	}
	return inst, nil
}

// UninstallInstallation 卸载安装
func (s *InstallationService) UninstallInstallation(ctx context.Context, requesterID string, installationID string) error {
	requesterUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(installationID)
	if err != nil {
		return err
	}

	inst, err := s.installationRepo.FindByIDWithApp(ctx, id)
	if err != nil {
		return errInstallationNotFound
	}

	if !s.canManage(ctx, inst, requesterUUID) {
		return errNotAuthorized
	}

	// 群聊安装:同时从会话移除 Bot 成员身份
	if inst.TargetType == models.InstallationTargetConversation {
		if err := s.enrollmentRepo.DeleteByConversationAndUser(ctx, inst.TargetID, inst.AppID); err != nil {
			logger.ErrorfWithCaller("[InstallationService] Failed to remove bot enrollment: %v", err)
		}
	}

	return s.installationRepo.Delete(ctx, id)
}

// canManage 判断请求者是否有权管理该安装
// user 安装:installer 本人;conversation 安装:installer、会话 owner/admin、Bot owner
func (s *InstallationService) canManage(ctx context.Context, inst *models.BotInstallation, requesterUUID uuid.UUID) bool {
	if inst.InstalledBy == requesterUUID {
		return true
	}
	if inst.App != nil && inst.App.OwnerID == requesterUUID {
		return true
	}
	if inst.TargetType == models.InstallationTargetConversation {
		enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, inst.TargetID, requesterUUID)
		if err == nil && enrollment != nil {
			return enrollment.Role == models.EnrollmentRoleOwner || enrollment.Role == models.EnrollmentRoleAdmin
		}
	}
	return false
}

// notifyExternalBotInstalled 声明 network:external 的 Bot 安装到群聊后,发系统消息 + WS 通知告知外发
func (s *InstallationService) notifyExternalBotInstalled(ctx context.Context, bot *models.Bot, conversationID uuid.UUID, installerID uuid.UUID) {
	sysContent := &models.SystemMessageContent{
		Type:    "bot_external_warning",
		BotID:   bot.ID.String(),
		BotName: bot.Name,
	}
	sysJSON, _ := json.Marshal(sysContent)
	sysMessage := &models.Message{
		SenderID: bot.ID,
		Content:  string(sysJSON),
		MsgType:  models.MsgTypeSystem,
	}
	if err := s.messageRepo.InsertMessage(ctx, conversationID, sysMessage); err != nil {
		logger.ErrorfWithCaller("[InstallationService] Failed to insert external warning system message: %v", err)
	}

	if websocket.GlobalHub != nil {
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			for _, m := range members {
				websocket.GlobalHub.SendToUser(m.UserID, "bot_external_warning", map[string]any{
					"bot_id":          bot.ID.String(),
					"bot_name":        bot.Name,
					"conversation_id": conversationID.String(),
					"installed_by":    installerID.String(),
				})
			}
		}
	}
}
