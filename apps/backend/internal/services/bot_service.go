package services

import (
    "context"
    "encoding/json"
    "errors"
    "time"

    "purr-chat-server/internal/botengine"
    "purr-chat-server/internal/models"
    "purr-chat-server/internal/repository"
    "purr-chat-server/internal/websocket"
    "purr-chat-server/pkg/database"
    "purr-chat-server/pkg/logger"

    "github.com/google/uuid"
)

// BotService Bot 业务逻辑服务
type BotService struct {
    botRepo         repository.BotRepository
    botDeployRepo   repository.BotDeploymentRepository
    userRepo        repository.UserRepository
    conversationRepo repository.ConversationRepository
    enrollmentRepo  repository.EnrollmentRepository
    messageRepo     repository.ConversationMessageRepository
}

// NewBotService 创建 Bot 服务
func NewBotService(
    botRepo repository.BotRepository,
    botDeployRepo repository.BotDeploymentRepository,
    userRepo repository.UserRepository,
    conversationRepo repository.ConversationRepository,
    enrollmentRepo repository.EnrollmentRepository,
    messageRepo repository.ConversationMessageRepository,
) *BotService {
    return &BotService{
        botRepo:         botRepo,
        botDeployRepo:   botDeployRepo,
        userRepo:        userRepo,
        conversationRepo: conversationRepo,
        enrollmentRepo:  enrollmentRepo,
        messageRepo:     messageRepo,
    }
}

// CreateBot 创建 Bot
func (s *BotService) CreateBot(ctx context.Context, ownerID string, req *models.CreateBotRequest) (*models.Bot, error) {
    ownerUUID, err := uuid.Parse(ownerID)
    if err != nil {
        return nil, err
    }

    visibility := req.Visibility
    if visibility == "" {
        visibility = models.BotVisibilityPrivate
    }

    bot := &models.Bot{
        OwnerID:         ownerUUID,
        Name:            req.Name,
        AvatarURL:       req.AvatarURL,
        Description:     req.Description,
        Status:          models.BotStatusActive,
        Visibility:      visibility,
        MechanismConfig: botengine.DefaultMechanismConfig(),
    }

    err = s.botRepo.Create(ctx, bot)
    if err != nil {
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

// SearchPublicBots 搜索公开 Bot
func (s *BotService) SearchPublicBots(ctx context.Context, query string, limit, offset int) ([]*models.Bot, error) {
    return s.botRepo.FindPublic(ctx, query, limit, offset)
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

    // 查询用户所在的群聊，排除 Bot 已部署的
    query := `
        SELECT c.id, c.name, COUNT(e.id) AS member_count
        FROM conversations c
        JOIN enrollments e ON e.conversation_id = c.id
        WHERE e.user_id = $1
          AND c.conversation_type = 'group'
          AND c.id NOT IN (
              SELECT conversation_id FROM bot_deployments WHERE bot_id = $2
          )
        GROUP BY c.id, c.name
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
        if err := rows.Scan(&dc.ID, &dc.Name, &dc.MemberCount); err != nil {
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
    if req.TriggerConfig != nil {
        bot.TriggerConfig = req.TriggerConfig
    }
    if req.ReplyConfig != nil {
        bot.ReplyConfig = req.ReplyConfig
    }
    if req.SpecialModeConfig != nil {
        bot.SpecialModeConfig = req.SpecialModeConfig
    }
    if req.MechanismConfig != nil {
        bot.MechanismConfig = req.MechanismConfig
    }

    err = s.botRepo.Update(ctx, bot)
    if err != nil {
        return nil, err
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

    return s.botRepo.Delete(ctx, id)
}

// DeployBot 将 Bot 部署到会话
func (s *BotService) DeployBot(ctx context.Context, botID, userID string, req *models.DeployBotRequest) (*models.BotDeployment, error) {
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

    // 检查是否已部署
    _, err = s.botDeployRepo.FindByBotAndConversation(ctx, botUUID, req.ConversationID)
    if err == nil {
        return nil, errors.New("bot already deployed to this conversation")
    }

    // 创建部署
    deployment := &models.BotDeployment{
        BotID:          botUUID,
        ConversationID: req.ConversationID,
        DeployedBy:     userUUID,
        Status:         models.BotDeploymentActive,
    }

    err = s.botDeployRepo.Create(ctx, deployment)
    if err != nil {
        return nil, err
    }

    // 标记会话有 Bot
    _, _ = database.GetPool().Exec(ctx, "UPDATE conversations SET bot_enabled = TRUE WHERE id = $1", req.ConversationID)

    // 插入系统消息：Bot 已加入对话
    sysContent := &models.SystemMessageContent{
        Type:    "bot_deployed",
        BotID:   botID,
        BotName: bot.Name,
    }
    sysJSON, _ := json.Marshal(sysContent)
    sysMessage := &models.Message{
        SenderID: uuid.Nil,
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

    logger.InfofWithCaller("Bot %s deployed to conversation %s by user %s", botID, req.ConversationID.String(), userID)

    return deployment, nil
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

    // 验证部署存在
    deployment, err := s.botDeployRepo.FindByBotAndConversation(ctx, botUUID, convUUID)
    if err != nil {
        return errors.New("bot not deployed to this conversation")
    }

    // 验证权限：部署者或 Bot owner 可以移除
    userUUID, _ := uuid.Parse(userID)
    if deployment.DeployedBy != userUUID {
        bot, err := s.botRepo.FindByID(ctx, botUUID)
        if err != nil || bot.OwnerID != userUUID {
            return errors.New("not authorized")
        }
    }

    err = s.botDeployRepo.DeleteByBotAndConversation(ctx, botUUID, convUUID)
    if err != nil {
        return err
    }

    // 获取 Bot 名称用于系统消息
    botName := "Bot"
    if bot, err := s.botRepo.FindByID(ctx, botUUID); err == nil {
        botName = bot.Name
    }

    // 插入系统消息：Bot 已离开对话
    undeploySysContent := &models.SystemMessageContent{
        Type:    "bot_undeployed",
        BotID:   botID,
        BotName: botName,
    }
    undeploySysJSON, _ := json.Marshal(undeploySysContent)
    undeploySysMessage := &models.Message{
        SenderID: uuid.Nil,
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

// GetBotDeployments 获取用户可见的 Bot 部署列表
func (s *BotService) GetBotDeployments(ctx context.Context, userID string) ([]*models.BotDeployment, error) {
    id, err := uuid.Parse(userID)
    if err != nil {
        return nil, err
    }

    return s.botDeployRepo.FindByUser(ctx, id)
}

// UpdateDeploymentStatus 更新部署状态（暂停/恢复）
func (s *BotService) UpdateDeploymentStatus(ctx context.Context, botID, userID string, req *models.UpdateDeploymentStatusRequest) error {
    botUUID, _ := uuid.Parse(botID)
    convUUID := req.ConversationID

    deployment, err := s.botDeployRepo.FindByBotAndConversation(ctx, botUUID, convUUID)
    if err != nil {
        return errors.New("deployment not found")
    }

    deployment.Status = models.BotDeploymentStatus(req.Status)
    return s.botDeployRepo.Update(ctx, deployment)
}

// ActivateSpecialMode 激活特殊模式
func (s *BotService) ActivateSpecialMode(ctx context.Context, botID, userID string, conversationID uuid.UUID) error {
    userUUID, _ := uuid.Parse(userID)

    // 验证部署存在
    deployment, err := s.botDeployRepo.FindByBotAndConversation(ctx, uuid.MustParse(botID), conversationID)
    if err != nil {
        return errors.New("bot not deployed to this conversation")
    }

    // 验证权限：部署者或 Bot owner 可以激活
    if deployment.DeployedBy != userUUID {
        bot, err := s.botRepo.FindByID(ctx, uuid.MustParse(botID))
        if err != nil || bot.OwnerID != userUUID {
            return errors.New("not authorized")
        }
    }

    // 检查该会话是否已有其他 Bot 的特殊模式活跃
    deployments, err := s.botDeployRepo.FindByConversationID(ctx, conversationID)
    if err == nil {
        for _, d := range deployments {
            if d.SpecialModeActive && d.BotID.String() != botID {
                return errors.New("another bot's special mode is already active in this conversation")
            }
        }
    }

    return nil // 实际激活由 BotEngine 处理
}

// DeactivateSpecialMode 停用特殊模式
func (s *BotService) DeactivateSpecialMode(ctx context.Context, botID, userID string, conversationID uuid.UUID) error {
    userUUID, _ := uuid.Parse(userID)

    // 验证部署存在
    deployment, err := s.botDeployRepo.FindByBotAndConversation(ctx, uuid.MustParse(botID), conversationID)
    if err != nil {
        return errors.New("bot not deployed to this conversation")
    }

    // 验证权限
    if deployment.DeployedBy != userUUID {
        bot, err := s.botRepo.FindByID(ctx, uuid.MustParse(botID))
        if err != nil || bot.OwnerID != userUUID {
            return errors.New("not authorized")
        }
    }

    return nil // 实际停用由 BotEngine 处理
}

// CreateBotConversation 创建与 Bot 的私聊会话
func (s *BotService) CreateBotConversation(ctx context.Context, botID, userID string) (*models.Conversation, error) {
    userUUID, err := uuid.Parse(userID)
    if err != nil {
        return nil, err
    }

    // 验证 Bot 存在且状态正常
    _, err = s.botRepo.FindByID(ctx, uuid.MustParse(botID))
    if err != nil {
        return nil, errors.New("bot not found")
    }

    // 创建会话（使用一个特殊的占位逻辑）
    // Bot 不是 users 表中的用户，所以我们需要一种方式来表示 Bot 会话
    // 方案：创建一个以 user 为 owner 的 direct 会话，Bot 的部署记录关联到此会话
    conversation := &models.Conversation{
        ConversationType: models.ConversationTypeDirect,
        Name:             "", // 后续动态设置
        CreatedBy:        &userUUID,
    }

    err = s.conversationRepo.Create(ctx, conversation)
    if err != nil {
        return nil, err
    }

    // 创建用户 enrollment
    userEnrollment := &models.Enrollment{
        ConversationID: conversation.ID,
        UserID:         userUUID,
        Role:           models.EnrollmentRoleOwner,
        JoinedAt:       time.Now().UTC(),
    }
    if err := s.enrollmentRepo.Create(ctx, userEnrollment); err != nil {
        logger.ErrorfWithCaller("[BotService] Failed to enroll bot owner: %v", err)
    }

    // 自动部署 Bot 到此会话
    _, err = s.DeployBot(ctx, botID, userID, &models.DeployBotRequest{
        ConversationID: conversation.ID,
    })
    if err != nil {
        // 部署失败不影响会话创建
        logger.ErrorfWithCaller("Failed to auto-deploy bot %s to conversation %s: %v", botID, conversation.ID, err)
    }

    return conversation, nil
}

// GetActiveBotsForConversation 获取会话中活跃的 Bot 列表（含 Bot 信息）
func (s *BotService) GetActiveBotsForConversation(ctx context.Context, conversationID string) ([]*models.BotDeployment, error) {
    convUUID, err := uuid.Parse(conversationID)
    if err != nil {
        return nil, err
    }

    deployments, err := s.botDeployRepo.FindActiveByConversation(ctx, convUUID)
    if err != nil {
        return nil, err
    }

    // 为每个部署填充 Bot 信息
    for _, dep := range deployments {
        bot, err := s.botRepo.FindByID(ctx, dep.BotID)
        if err == nil {
            dep.Bot = bot
        }
    }

    return deployments, nil
}
