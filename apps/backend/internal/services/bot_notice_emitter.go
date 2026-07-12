package services

import (
	"context"
	"sync/atomic"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

type NoticeMetrics struct {
	MemberSent          atomic.Int64
	MemberSkipped       atomic.Int64
	InstallationSent    atomic.Int64
	InstallationSkipped atomic.Int64
}

type NoticeMetricsSnapshot struct {
	MemberSent          int64
	MemberSkipped       int64
	InstallationSent    int64
	InstallationSkipped int64
}

type BotNoticeEmitter struct {
	installationRepo repository.BotInstallationRepository
	botRepo          repository.BotRepository
	publisher        BotEventPublisher
	metrics          NoticeMetrics
	now              func() time.Time
}

func NewBotNoticeEmitter(
	installationRepo repository.BotInstallationRepository,
	botRepo repository.BotRepository,
	publisher BotEventPublisher,
) *BotNoticeEmitter {
	return &BotNoticeEmitter{
		installationRepo: installationRepo,
		botRepo:          botRepo,
		publisher:        publisher,
		now:              time.Now,
	}
}

func (e *BotNoticeEmitter) MetricsSnapshot() NoticeMetricsSnapshot {
	return NoticeMetricsSnapshot{
		MemberSent:          e.metrics.MemberSent.Load(),
		MemberSkipped:       e.metrics.MemberSkipped.Load(),
		InstallationSent:    e.metrics.InstallationSent.Load(),
		InstallationSkipped: e.metrics.InstallationSkipped.Load(),
	}
}

func (e *BotNoticeEmitter) NotifyMemberJoined(ctx context.Context, convID, userID uuid.UUID, role string) {
	e.emitMemberNotice(ctx, convID, onebot.NoticeGroupMemberIncrease, map[string]any{
		"conversation_id": convID.String(),
		"user_id":         userID.String(),
		"role":            role,
	})
}

func (e *BotNoticeEmitter) NotifyMemberLeft(ctx context.Context, convID, userID uuid.UUID) {
	e.emitMemberNotice(ctx, convID, onebot.NoticeGroupMemberDecrease, map[string]any{
		"conversation_id": convID.String(),
		"user_id":         userID.String(),
	})
}

func (e *BotNoticeEmitter) NotifyMemberRoleChanged(ctx context.Context, convID, userID uuid.UUID, oldRole, newRole string) {
	e.emitMemberNotice(ctx, convID, onebot.NoticeGroupMemberRoleChanged, map[string]any{
		"conversation_id": convID.String(),
		"user_id":         userID.String(),
		"old_role":        oldRole,
		"new_role":        newRole,
	})
}

func (e *BotNoticeEmitter) emitMemberNotice(ctx context.Context, convID uuid.UUID, detailType string, data map[string]any) {
	if e == nil || e.publisher == nil {
		return
	}

	installations, err := e.installationRepo.FindActiveByConversation(ctx, convID)
	if err != nil {
		e.metrics.MemberSkipped.Add(1)
		logger.ErrorfWithCaller("[BotNoticeEmitter] Failed to find active installations for conversation %s: %v", convID, err)
		return
	}

	for _, inst := range installations {
		if !models.HasCapability(inst.GrantedCapabilities, models.CapabilityMembersRead) {
			continue
		}

		bot, err := e.botRepo.FindByID(ctx, inst.AppID)
		if err != nil || bot.Status != models.BotStatusActive {
			e.metrics.MemberSkipped.Add(1)
			continue
		}

		event, err := onebot.BuildNoticeEvent(bot.ID.String(), detailType, "", e.now(), data)
		if err != nil {
			e.metrics.MemberSkipped.Add(1)
			continue
		}

		delivered := e.publisher.PublishBotEvent(bot.ID, event)
		if delivered > 0 {
			e.metrics.MemberSent.Add(1)
			logger.InfofWithCaller("[BotNoticeEmitter] Pushed %s notice to bot %s (%d connections)",
				detailType, bot.Name, delivered)
		} else {
			e.metrics.MemberSkipped.Add(1)
		}
	}
}

func (e *BotNoticeEmitter) NotifyInstallationInstalled(ctx context.Context, inst *models.BotInstallation) {
	e.emitInstallationNotice(ctx, inst, onebot.SubTypeInstalled)
}

func (e *BotNoticeEmitter) NotifyInstallationSuspended(ctx context.Context, inst *models.BotInstallation) {
	e.emitInstallationNotice(ctx, inst, onebot.SubTypeSuspended)
}

func (e *BotNoticeEmitter) NotifyInstallationResumed(ctx context.Context, inst *models.BotInstallation) {
	e.emitInstallationNotice(ctx, inst, onebot.SubTypeResumed)
}

func (e *BotNoticeEmitter) NotifyInstallationUninstalled(ctx context.Context, inst *models.BotInstallation) {
	e.emitInstallationNotice(ctx, inst, onebot.SubTypeUninstalled)
}

func (e *BotNoticeEmitter) NotifyInstallationCapabilityChanged(ctx context.Context, inst *models.BotInstallation) {
	e.emitInstallationNotice(ctx, inst, onebot.SubTypeCapabilityChanged)
}

func (e *BotNoticeEmitter) emitInstallationNotice(ctx context.Context, inst *models.BotInstallation, subType string) {
	if e == nil || e.publisher == nil {
		return
	}

	bot, err := e.botRepo.FindByID(ctx, inst.AppID)
	if err != nil || bot.Status != models.BotStatusActive {
		e.metrics.InstallationSkipped.Add(1)
		return
	}

	data := map[string]any{
		"installation_id":      inst.ID.String(),
		"target_type":          string(inst.TargetType),
		"target_id":            inst.TargetID.String(),
		"status":               string(inst.Status),
		"granted_capabilities": inst.GrantedCapabilities,
		"change_type":          subType,
	}

	event, err := onebot.BuildNoticeEvent(bot.ID.String(), onebot.NoticeInstallationChanged, subType, e.now(), data)
	if err != nil {
		e.metrics.InstallationSkipped.Add(1)
		return
	}

	delivered := e.publisher.PublishBotEvent(bot.ID, event)
	if delivered > 0 {
		e.metrics.InstallationSent.Add(1)
		logger.InfofWithCaller("[BotNoticeEmitter] Pushed installation %s notice to bot %s (%d connections)",
			subType, bot.Name, delivered)
	} else {
		e.metrics.InstallationSkipped.Add(1)
	}
}
