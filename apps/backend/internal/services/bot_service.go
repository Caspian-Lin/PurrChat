package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"purr-chat-server/internal/botengine"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BotService Bot 业务逻辑服务
type BotService struct {
	botRepo          repository.BotRepository
	installationRepo repository.BotInstallationRepository
	userRepo         repository.UserRepository
	friendshipRepo   repository.FriendshipRepository
	conversationRepo repository.ConversationRepository
	enrollmentRepo   repository.EnrollmentRepository
	messageRepo      repository.ConversationMessageRepository
	callLogRepo      repository.BotCallLogRepository
	connections      BotConnectionCloser
}

type BotConnectionCloser interface {
	DisconnectBot(context.Context, uuid.UUID) error
}

type NoopBotConnectionCloser struct{}

func (NoopBotConnectionCloser) DisconnectBot(context.Context, uuid.UUID) error { return nil }

// NewBotService 创建 Bot 服务
func NewBotService(
	botRepo repository.BotRepository,
	installationRepo repository.BotInstallationRepository,
	userRepo repository.UserRepository,
	friendshipRepo repository.FriendshipRepository,
	conversationRepo repository.ConversationRepository,
	enrollmentRepo repository.EnrollmentRepository,
	messageRepo repository.ConversationMessageRepository,
	callLogRepo repository.BotCallLogRepository,
) *BotService {
	return &BotService{
		botRepo:          botRepo,
		installationRepo: installationRepo,
		userRepo:         userRepo,
		friendshipRepo:   friendshipRepo,
		conversationRepo: conversationRepo,
		enrollmentRepo:   enrollmentRepo,
		messageRepo:      messageRepo,
		callLogRepo:      callLogRepo,
		connections:      NoopBotConnectionCloser{},
	}
}

func (s *BotService) SetConnectionCloser(closer BotConnectionCloser) {
	if closer == nil {
		closer = NoopBotConnectionCloser{}
	}
	s.connections = closer
}

func (s *BotService) OwnsBot(ctx context.Context, ownerID, botID uuid.UUID) (bool, error) {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return bot.OwnerID == ownerID, nil
}

// deriveDiagnosticsConsent 根据 requested_capabilities 推导 diagnostics_consent
// 声明 network:external 的 Bot 强制 granted(数据已必然到达 owner 经第三方)
func deriveDiagnosticsConsent(requestedCapabilities []string) models.DiagnosticsConsent {
	if models.HasCapability(requestedCapabilities, models.CapabilityNetworkExternal) {
		return models.DiagnosticsGranted
	}
	return models.DiagnosticsDenied
}

// CreateBot 创建 Bot
func (s *BotService) CreateBot(ctx context.Context, ownerID string, req *models.CreateBotRequest) (*models.Bot, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, err
	}

	// 可见性:优先用 discoverability;兼容旧 visibility 字段映射
	visibility := req.Visibility
	if visibility == "" {
		visibility = models.BotVisibilityPrivate
	}
	discoverability := req.Discoverability
	if discoverability == "" {
		switch visibility {
		case models.BotVisibilityPublic, models.BotVisibilityGlobal:
			discoverability = models.DiscoverabilityListed
		default:
			discoverability = models.DiscoverabilityUnlisted
		}
	}

	bot := &models.Bot{
		OwnerID:         ownerUUID,
		Name:            req.Name,
		AvatarURL:       req.AvatarURL,
		Description:     req.Description,
		Status:          models.BotStatusActive,
		Visibility:      visibility,
		Discoverability: discoverability,
		IsSystem:        visibility == models.BotVisibilityGlobal,
		BotType:         models.BotTypeWorkflow,
		MechanismConfig: botengine.DefaultMechanismConfig(),
	}

	// 在共享事务中创建 Bot、会话、enrollment 和 installation，防止半安装状态。
	err = pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := s.botRepo.CreateTx(ctx, tx, bot); err != nil {
			return fmt.Errorf("create bot: %w", err)
		}

		// 自动创建 owner ↔ bot 的私聊会话
		conversation := &models.Conversation{
			ConversationType: models.ConversationTypeDirect,
			CreatedBy:        &ownerUUID,
		}
		if err := s.conversationRepo.CreateTx(ctx, tx, conversation); err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}

		ownerEnrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         ownerUUID,
			Role:           models.EnrollmentRoleOwner,
			JoinedAt:       time.Now().UTC(),
		}
		if err := s.enrollmentRepo.CreateTx(ctx, tx, ownerEnrollment); err != nil {
			return fmt.Errorf("create owner enrollment: %w", err)
		}

		botEnrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         bot.ID,
			Role:           models.EnrollmentRoleMember,
			JoinedAt:       time.Now().UTC(),
		}
		if err := s.enrollmentRepo.CreateTx(ctx, tx, botEnrollment); err != nil {
			return fmt.Errorf("create bot enrollment: %w", err)
		}

		// 自动为 owner 创建 user 安装
		installation := &models.BotInstallation{
			AppID:               bot.ID,
			InstalledBy:         ownerUUID,
			TargetType:          models.InstallationTargetUser,
			TargetID:            ownerUUID,
			GrantedCapabilities: bot.RequestedCapabilities,
			DiagnosticsConsent:  models.DiagnosticsGranted,
			Status:              models.InstallationActive,
		}
		if err := s.installationRepo.CreateTx(ctx, tx, installation); err != nil {
			return fmt.Errorf("create owner installation: %w", err)
		}

		return nil
	})
	if err != nil {
		logger.ErrorfWithCaller("[BotService] Failed to create bot conversation/installation (transaction rolled back): %v", err)
		return nil, err
	}

	return bot, nil
}

// GetBot 获取 Bot 详情
func (s *BotService) GetBot(ctx context.Context, botID string) (*models.Bot, error) {
	id, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	return s.botRepo.FindByID(ctx, id)
}

// ListBots 获取用户创建的 Bot 列表
func (s *BotService) ListBots(ctx context.Context, ownerID string) ([]*models.Bot, error) {
	id, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, err
	}

	return s.botRepo.FindByOwner(ctx, id)
}

// SearchPublicBotsPaginated 分页搜索公开 Bot（含统计信息）
func (s *BotService) SearchPublicBotsPaginated(ctx context.Context, query string, limit, offset int) (*models.PaginatedSearchResult, error) {
	bots, err := s.botRepo.FindPublicWithDetails(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.botRepo.CountPublic(ctx, query)
	if err != nil {
		return nil, err
	}

	// 填充 trigger_summary 和 reply_type（从 mechanism_config 中提取）
	for _, bot := range bots {
		mc, err := botengine.ParseMechanismConfig(bot.MechanismConfig)
		if err == nil && len(mc.Mechanisms) > 0 {
			// 使用第一个启用的机制生成摘要
			for _, mech := range mc.Mechanisms {
				if mech.Enabled {
					triggerSummary, replySummary := botengine.GetMechanismSummary(mech)
					bot.TriggerSummary = triggerSummary
					bot.ReplyType = replySummary
					break
				}
			}
		}
	}

	return &models.PaginatedSearchResult{
		Bots:   bots,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// GetDeployableConversations 获取用户可部署 Bot 的群聊列表
func (s *BotService) GetDeployableConversations(ctx context.Context, userID string, botID string) ([]*models.DeployableConversation, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	// 查询用户所在的群聊，排除 Bot 已部署的。私聊不需要安装
	// （创建/添加 Bot 时自动建立 user installation），只有群聊需要显式安装。
	query := `
        SELECT c.id, c.name, c.conversation_type, c.avatar_url, COUNT(e.id) AS member_count
        FROM conversations c
        JOIN enrollments e ON e.conversation_id = c.id
        WHERE e.user_id = $1
          AND c.conversation_type = 'group'
          AND c.id NOT IN (
              SELECT target_id FROM bot_installations WHERE target_type = 'conversation' AND app_id = $2
          )
        GROUP BY c.id, c.name, c.conversation_type, c.avatar_url
        ORDER BY c.updated_at DESC
    `

	rows, err := database.GetPool().Query(ctx, query, userUUID, botUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.DeployableConversation
	for rows.Next() {
		dc := &models.DeployableConversation{}
		if err := rows.Scan(&dc.ID, &dc.Name, &dc.ConversationType, &dc.AvatarURL, &dc.MemberCount); err != nil {
			return nil, err
		}
		results = append(results, dc)
	}

	return results, nil
}

// UpdateBot 更新 Bot 配置
func (s *BotService) UpdateBot(ctx context.Context, botID string, userID string, req *models.UpdateBotRequest) (*models.Bot, error) {
	id, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	bot, err := s.botRepo.FindByID(ctx, id)
	if err != nil {
		return nil, errors.New("bot not found")
	}

	// 验证所有权
	userUUID, _ := uuid.Parse(userID)
	if bot.OwnerID != userUUID {
		return nil, errors.New("not the bot owner")
	}

	// 更新字段
	if req.Name != "" {
		bot.Name = req.Name
	}
	if req.AvatarURL != "" {
		bot.AvatarURL = req.AvatarURL
	}
	if req.Description != "" {
		bot.Description = req.Description
	}
	if req.Status != "" {
		bot.Status = req.Status
	}
	if req.Visibility != "" {
		bot.Visibility = req.Visibility
	}
	if req.MechanismConfig != nil {
		bot.MechanismConfig = req.MechanismConfig
	}
	if req.RequestedCapabilities != nil {
		bot.RequestedCapabilities = req.RequestedCapabilities
	}
	if req.AllowedEndpoints != nil {
		bot.AllowedEndpoints = req.AllowedEndpoints
	}

	err = s.botRepo.Update(ctx, bot)
	if err != nil {
		return nil, err
	}
	// 显式更新权限时同步 owner 安装；mechanism_config 不再自动推导可执行权限（#87）
	if req.RequestedCapabilities != nil {
		if _, syncErr := database.GetPool().Exec(ctx, `
			UPDATE bot_installations
			SET granted_capabilities = $1,
				diagnostics_consent = CASE
					WHEN $1::text[] @> ARRAY['network:external']::text[] THEN 'granted'
					ELSE diagnostics_consent
				END,
				updated_at = NOW()
			WHERE app_id = $2 AND installed_by = $3
		`, bot.RequestedCapabilities, bot.ID, userUUID); syncErr != nil {
			return nil, fmt.Errorf("sync owner installation capabilities: %w", syncErr)
		}
	}
	if bot.Status == models.BotStatusDisabled {
		_ = s.connections.DisconnectBot(ctx, bot.ID)
	}

	return bot, nil
}

// DeleteBot 删除 Bot
func (s *BotService) DeleteBot(ctx context.Context, botID string, userID string) error {
	id, err := uuid.Parse(botID)
	if err != nil {
		return err
	}

	bot, err := s.botRepo.FindByID(ctx, id)
	if err != nil {
		return errors.New("bot not found")
	}

	userUUID, _ := uuid.Parse(userID)
	if bot.OwnerID != userUUID {
		return errors.New("not the bot owner")
	}

	if err := s.botRepo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.connections.DisconnectBot(ctx, id)
	return nil
}

// DeployBot 将 Bot 部署到会话
func (s *BotService) DeployBot(ctx context.Context, botID, userID string, req *models.DeployBotRequest) (*models.BotInstallation, error) {
	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	// 验证 Bot 存在且状态正常
	bot, err := s.botRepo.FindByID(ctx, botUUID)
	if err != nil {
		return nil, errors.New("bot not found")
	}

	if bot.Status != models.BotStatusActive {
		return nil, errors.New("bot is not active")
	}

	// 验证可见性：private 只能部署到自己的会话
	if bot.Visibility == models.BotVisibilityPrivate && bot.OwnerID != userUUID {
		return nil, errors.New("this bot is private")
	}

	// 验证用户是会话成员
	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, req.ConversationID, userUUID)
	if err != nil {
		return nil, errors.New("not a participant in this conversation")
	}

	// 检查 Bot 是否已是 enrollment 成员
	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, req.ConversationID, botUUID)
	if err == nil {
		return nil, errors.New("bot is already a member of this conversation")
	}

	// 在共享事务中创建 enrollment 和 installation
	installation := &models.BotInstallation{
		AppID:               botUUID,
		InstalledBy:         userUUID,
		TargetType:          models.InstallationTargetConversation,
		TargetID:            req.ConversationID,
		GrantedCapabilities: bot.RequestedCapabilities,
		DiagnosticsConsent:  deriveDiagnosticsConsent(bot.RequestedCapabilities),
		Status:              models.InstallationActive,
	}

	botEnrollment := &models.Enrollment{
		ConversationID: req.ConversationID,
		UserID:         botUUID,
		Role:           models.EnrollmentRoleMember,
		JoinedAt:       time.Now().UTC(),
	}

	err = pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := s.enrollmentRepo.CreateTx(ctx, tx, botEnrollment); err != nil {
			return fmt.Errorf("create bot enrollment: %w", err)
		}
		if err := s.installationRepo.CreateTx(ctx, tx, installation); err != nil {
			return fmt.Errorf("create installation: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to deploy bot (transaction rolled back): %w", err)
	}

	// 插入系统消息：Bot 已加入对话（sender_id = bot 的 user_id）
	sysContent := &models.SystemMessageContent{
		Type:    "bot_deployed",
		BotID:   botID,
		BotName: bot.Name,
	}
	sysJSON, _ := json.Marshal(sysContent)
	sysMessage := &models.Message{
		SenderID: bot.ID, // Bot 现在是真实用户
		Content:  string(sysJSON),
		MsgType:  models.MsgTypeSystem,
	}
	if insertErr := s.messageRepo.InsertMessage(ctx, req.ConversationID, sysMessage); insertErr != nil {
		logger.ErrorfWithCaller("[BotService] Failed to insert system message for bot deploy: %v", insertErr)
	}

	// WebSocket 通知会话成员
	if websocket.GlobalHub != nil {
		members, err := s.enrollmentRepo.FindByConversationID(ctx, req.ConversationID)
		if err == nil {
			for _, m := range members {
				websocket.GlobalHub.SendToUser(m.UserID, "bot_deployed", map[string]any{
					"bot_id":          botID,
					"bot_name":        bot.Name,
					"conversation_id": req.ConversationID.String(),
					"deployed_by":     userID,
				})
			}
		}
	}

	logger.InfofWithCaller("Bot %s installed to conversation %s by user %s", botID, req.ConversationID.String(), userID)

	return installation, nil
}

// UndeployBot 从会话移除 Bot
func (s *BotService) UndeployBot(ctx context.Context, botID, conversationID, userID string) error {
	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return err
	}

	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		return err
	}

	// 获取 Bot 信息（用于系统消息和权限验证）
	bot, err := s.botRepo.FindByID(ctx, botUUID)
	if err != nil {
		return errors.New("bot not found")
	}

	// 验证权限：Bot owner 可以移除，或会话管理员/群主可以移除
	userUUID, _ := uuid.Parse(userID)
	if bot.OwnerID != userUUID {
		// 检查操作者是否是会话管理员/群主
		operatorEnrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, convUUID, userUUID)
		if err != nil || (operatorEnrollment.Role != models.EnrollmentRoleAdmin && operatorEnrollment.Role != models.EnrollmentRoleOwner) {
			return errors.New("not authorized")
		}
	}

	// 移除 Bot 的 enrollment
	if err := s.enrollmentRepo.DeleteByConversationAndUser(ctx, convUUID, botUUID); err != nil {
		logger.ErrorfWithCaller("[BotService] Failed to remove bot enrollment: %v", err)
	}

	// 删除 installation 记录
	if err := s.installationRepo.DeleteByAppAndTarget(ctx, botUUID, models.InstallationTargetConversation, convUUID); err != nil {
		logger.ErrorfWithCaller("[BotService] Failed to remove installation record: %v", err)
	}

	// 插入系统消息：Bot 已离开对话（sender_id = bot 的 user_id）
	undeploySysContent := &models.SystemMessageContent{
		Type:    "bot_undeployed",
		BotID:   botID,
		BotName: bot.Name,
	}
	undeploySysJSON, _ := json.Marshal(undeploySysContent)
	undeploySysMessage := &models.Message{
		SenderID: bot.ID, // Bot 现在是真实用户
		Content:  string(undeploySysJSON),
		MsgType:  models.MsgTypeSystem,
	}
	if insertErr := s.messageRepo.InsertMessage(ctx, convUUID, undeploySysMessage); insertErr != nil {
		logger.ErrorfWithCaller("[BotService] Failed to insert system message for bot undeploy: %v", insertErr)
	}

	// WebSocket 通知
	if websocket.GlobalHub != nil {
		members, err := s.enrollmentRepo.FindByConversationID(ctx, convUUID)
		if err == nil {
			for _, m := range members {
				websocket.GlobalHub.SendToUser(m.UserID, "bot_undeployed", map[string]any{
					"bot_id":          botID,
					"conversation_id": conversationID,
				})
			}
		}
	}

	logger.InfofWithCaller("Bot %s undeployed from conversation %s", botID, conversationID)

	return nil
}

// GetBotDeployments 获取用户可见的 Bot 安装列表
func (s *BotService) GetBotDeployments(ctx context.Context, userID string) ([]*models.BotInstallation, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	installations, err := s.installationRepo.FindByInstaller(ctx, id)
	if err != nil {
		return nil, err
	}

	// 批量补充目标会话名与类型（仅 conversation 类型）
	convIDs := make([]uuid.UUID, 0, len(installations))
	for _, inst := range installations {
		if inst.TargetType == models.InstallationTargetConversation {
			convIDs = append(convIDs, inst.TargetID)
		}
	}
	if len(convIDs) > 0 {
		rows, qErr := database.GetPool().Query(ctx,
			`SELECT id, name, conversation_type FROM conversations WHERE id = ANY($1)`, convIDs)
		if qErr == nil {
			defer rows.Close()
			convInfo := make(map[uuid.UUID]struct{ name, convType string }, len(convIDs))
			for rows.Next() {
				var cID uuid.UUID
				var cName, cType string
				if sErr := rows.Scan(&cID, &cName, &cType); sErr == nil {
					convInfo[cID] = struct{ name, convType string }{cName, cType}
				}
			}
			for _, inst := range installations {
				if info, ok := convInfo[inst.TargetID]; ok {
					inst.TargetName = info.name
					inst.TargetConvType = info.convType
				}
			}
		}
	}

	// 批量补充 Bot 信息（供前端区分自己创建的 vs 已安装的公开 Bot）
	appIDSet := make(map[uuid.UUID]bool, len(installations))
	for _, inst := range installations {
		appIDSet[inst.AppID] = true
	}
	if len(appIDSet) > 0 {
		appIDs := make([]uuid.UUID, 0, len(appIDSet))
		for id := range appIDSet {
			appIDs = append(appIDs, id)
		}
		botRows, bErr := database.GetPool().Query(ctx, `
			SELECT id, owner_id, name, avatar_url, description, status, visibility, mechanism_config,
			       bot_type, discoverability, is_system, requested_capabilities,
			       allowed_endpoints, created_at, updated_at
			FROM bots WHERE id = ANY($1)
		`, appIDs)
		if bErr == nil {
			botMap := make(map[uuid.UUID]*models.Bot, len(appIDs))
			for botRows.Next() {
				var b models.Bot
				if sErr := botRows.Scan(
					&b.ID, &b.OwnerID, &b.Name, &b.AvatarURL, &b.Description, &b.Status, &b.Visibility,
					&b.MechanismConfig, &b.BotType, &b.Discoverability, &b.IsSystem,
					&b.RequestedCapabilities, &b.AllowedEndpoints, &b.CreatedAt, &b.UpdatedAt,
				); sErr == nil {
					botMap[b.ID] = &b
				}
			}
			botRows.Close()
			for _, inst := range installations {
				if b, ok := botMap[inst.AppID]; ok {
					inst.App = b
				}
			}
		}
	}

	return installations, nil
}

// UpdateDeploymentStatus 更新安装状态（暂停/恢复）
func (s *BotService) UpdateDeploymentStatus(ctx context.Context, botID, userID string, req *models.UpdateDeploymentStatusRequest) error {
	botUUID, err := parseID(botID)
	if err != nil {
		return err
	}
	requesterUUID, err := parseID(userID)
	if err != nil {
		return err
	}
	convUUID := req.ConversationID

	inst, err := s.installationRepo.FindByAppAndTarget(ctx, botUUID, models.InstallationTargetConversation, convUUID)
	if err != nil {
		return ErrResourceNotFound
	}
	if inst.InstalledBy != requesterUUID {
		bot, botErr := s.botRepo.FindByID(ctx, botUUID)
		enrollment, enrollmentErr := s.enrollmentRepo.FindByConversationAndUser(ctx, convUUID, requesterUUID)
		isBotOwner := botErr == nil && bot.OwnerID == requesterUUID
		isConversationManager := enrollmentErr == nil && enrollment != nil &&
			(enrollment.Role == models.EnrollmentRoleOwner || enrollment.Role == models.EnrollmentRoleAdmin)
		if !isBotOwner && !isConversationManager {
			return ErrResourceNotFound
		}
	}

	inst.Status = models.InstallationStatus(req.Status)
	return s.installationRepo.Update(ctx, inst)
}

// ActivateWorkflow 激活工作流
func (s *BotService) ActivateWorkflow(ctx context.Context, botID, userID string, conversationID uuid.UUID) error {
	userUUID, _ := uuid.Parse(userID)

	// 验证安装存在
	inst, err := s.installationRepo.FindByAppAndTarget(ctx, uuid.MustParse(botID), models.InstallationTargetConversation, conversationID)
	if err != nil {
		return errors.New("bot not installed to this conversation")
	}

	// 验证权限：安装者或 Bot owner 可以激活
	if inst.InstalledBy != userUUID {
		bot, err := s.botRepo.FindByID(ctx, uuid.MustParse(botID))
		if err != nil || bot.OwnerID != userUUID {
			return errors.New("not authorized")
		}
	}

	// 检查该会话是否已有其他 Bot 的工作流活跃
	installations, err := s.installationRepo.FindActiveByConversation(ctx, conversationID)
	if err == nil {
		for _, i := range installations {
			if i.Status == models.InstallationActive && i.AppID.String() != botID {
				return errors.New("another bot's workflow is already active in this conversation")
			}
		}
	}

	return nil // 实际激活由 BotEngine 处理
}

// DeactivateWorkflow 停用工作流
func (s *BotService) DeactivateWorkflow(ctx context.Context, botID, userID string, conversationID uuid.UUID) error {
	userUUID, _ := uuid.Parse(userID)

	// 验证安装存在
	inst, err := s.installationRepo.FindByAppAndTarget(ctx, uuid.MustParse(botID), models.InstallationTargetConversation, conversationID)
	if err != nil {
		return errors.New("bot not installed to this conversation")
	}

	// 验证权限
	if inst.InstalledBy != userUUID {
		bot, err := s.botRepo.FindByID(ctx, uuid.MustParse(botID))
		if err != nil || bot.OwnerID != userUUID {
			return errors.New("not authorized")
		}
	}

	return nil // 实际停用由 BotEngine 处理
}

// CreateBotConversation 创建与 Bot 的私聊会话(幂等)
// 任何用户均可添加 listed/featured Bot;unlisted Bot 只有 owner 可以
func (s *BotService) CreateBotConversation(ctx context.Context, botID, userID string) (*models.Conversation, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	// 验证 Bot 存在
	bot, err := s.botRepo.FindByID(ctx, botUUID)
	if err != nil {
		return nil, errors.New("bot not found")
	}

	// 权限校验:unlisted Bot 只有 owner 可以添加
	if bot.OwnerID != userUUID && bot.Discoverability == models.DiscoverabilityUnlisted {
		return nil, errors.New("this bot is private")
	}

	// 查找已有的私聊会话(幂等)
	existingConv, err := s.conversationRepo.FindByUsers(ctx, userUUID, botUUID)
	if err == nil {
		// 确保 installation 存在(幂等)
		if err := s.ensureUserInstallation(ctx, bot, userUUID); err != nil {
			return nil, err
		}
		return existingConv, nil
	}

	// 不存在则创建新的私聊会话（在共享事务中创建会话、enrollment 和 installation）
	conversation := &models.Conversation{
		ConversationType: models.ConversationTypeDirect,
		CreatedBy:        &userUUID,
	}

	installation := &models.BotInstallation{
		AppID:               bot.ID,
		InstalledBy:         userUUID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            userUUID,
		GrantedCapabilities: bot.RequestedCapabilities,
		DiagnosticsConsent:  deriveDiagnosticsConsent(bot.RequestedCapabilities),
		Status:              models.InstallationActive,
	}

	err = pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := s.conversationRepo.CreateTx(ctx, tx, conversation); err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}

		ownerEnrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         userUUID,
			Role:           models.EnrollmentRoleOwner,
			JoinedAt:       time.Now().UTC(),
		}
		if err := s.enrollmentRepo.CreateTx(ctx, tx, ownerEnrollment); err != nil {
			return fmt.Errorf("create owner enrollment: %w", err)
		}

		botEnrollment := &models.Enrollment{
			ConversationID: conversation.ID,
			UserID:         botUUID,
			Role:           models.EnrollmentRoleMember,
			JoinedAt:       time.Now().UTC(),
		}
		if err := s.enrollmentRepo.CreateTx(ctx, tx, botEnrollment); err != nil {
			return fmt.Errorf("create bot enrollment: %w", err)
		}

		if err := s.installationRepo.CreateTx(ctx, tx, installation); err != nil {
			return fmt.Errorf("create user installation: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bot conversation (transaction rolled back): %w", err)
	}

	return conversation, nil
}

// ensureUserInstallation 幂等创建并验证 user 安装。
func (s *BotService) ensureUserInstallation(ctx context.Context, bot *models.Bot, userID uuid.UUID) error {
	installation := &models.BotInstallation{
		AppID:               bot.ID,
		InstalledBy:         userID,
		TargetType:          models.InstallationTargetUser,
		TargetID:            userID,
		GrantedCapabilities: bot.RequestedCapabilities,
		DiagnosticsConsent:  deriveDiagnosticsConsent(bot.RequestedCapabilities),
		Status:              models.InstallationActive,
	}
	if err := s.installationRepo.Create(ctx, installation); err != nil {
		return fmt.Errorf("create user installation: %w", err)
	}
	if installation.Status != models.InstallationActive {
		return errors.New("bot installation is not active")
	}
	return nil
}

// GetActiveBotsForConversation 获取会话中活跃的 Bot 安装列表（含 Bot 信息）
func (s *BotService) GetActiveBotsForConversation(ctx context.Context, requesterID, conversationID string) ([]*models.BotInstallation, error) {
	requesterUUID, err := parseID(requesterID)
	if err != nil {
		return nil, err
	}
	convUUID, err := parseID(conversationID)
	if err != nil {
		return nil, err
	}
	if err := requireConversationMember(ctx, s.enrollmentRepo, convUUID, requesterUUID); err != nil {
		return nil, err
	}

	installations, err := s.installationRepo.FindActiveByConversation(ctx, convUUID)
	if err != nil {
		return nil, err
	}

	// 填充 Bot 信息
	for _, inst := range installations {
		bot, err := s.botRepo.FindByID(ctx, inst.AppID)
		if err == nil {
			inst.App = bot
		}
	}

	return installations, nil
}

// GetBotCallLogs 获取 Bot 调用日志
func (s *BotService) GetBotCallLogs(ctx context.Context, botID string, userID string, limit, offset int) (*models.BotCallLogListResponse, error) {
	botUUID, err := uuid.Parse(botID)
	if err != nil {
		return nil, err
	}

	// 验证 Bot 存在
	bot, err := s.botRepo.FindByID(ctx, botUUID)
	if err != nil {
		return nil, errors.New("bot not found")
	}

	// 验证权限：只有 Bot owner 可以查看调用日志
	userUUID, _ := uuid.Parse(userID)
	if bot.OwnerID != userUUID {
		return nil, errors.New("not the bot owner")
	}

	logs, err := s.callLogRepo.FindAllByBotID(ctx, botUUID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.callLogRepo.CountByBotID(ctx, botUUID)
	if err != nil {
		return nil, err
	}

	return &models.BotCallLogListResponse{
		Logs:   logs,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
