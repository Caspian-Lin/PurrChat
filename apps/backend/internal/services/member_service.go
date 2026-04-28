package services

import (
	"context"
	"errors"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
)

// MemberService 成员管理服务
type MemberService struct {
	userRepo         repository.UserRepository
	conversationRepo repository.ConversationRepository
	enrollmentRepo   repository.EnrollmentRepository
}

// NewMemberService 创建成员管理服务
func NewMemberService(
	userRepo repository.UserRepository,
	conversationRepo repository.ConversationRepository,
	enrollmentRepo repository.EnrollmentRepository,
) *MemberService {
	return &MemberService{
		userRepo:         userRepo,
		conversationRepo: conversationRepo,
		enrollmentRepo:   enrollmentRepo,
	}
}

// AddMemberToConversation 添加成员到会话
func (s *MemberService) AddMemberToConversation(ctx context.Context, conversationIDStr, userID, targetUserID string, role models.EnrollmentRole) error {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return err
	}

	// 检查操作者是否是会话的管理员或拥有者
	enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, userUUID)
	if err != nil {
		return errors.New("not authorized")
	}

	if enrollment.Role != models.EnrollmentRoleOwner && enrollment.Role != models.EnrollmentRoleAdmin {
		return errors.New("not authorized")
	}

	// 检查目标用户是否已经在会话中
	_, err = s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, targetUUID)
	if err == nil {
		return errors.New("user already in conversation")
	}

	// 如果目标是 Bot，强制角色为 member
	targetUser, err := s.userRepo.FindByID(ctx, targetUUID)
	if err == nil && targetUser.IsBot {
		role = models.EnrollmentRoleMember
	}

	// 添加成员（使用UTC时间）
	newEnrollment := &models.Enrollment{
		ConversationID: conversationID,
		UserID:         targetUUID,
		Role:           role,
		JoinedAt:       time.Now().UTC(),
	}

	err = s.enrollmentRepo.Create(ctx, newEnrollment)
	if err != nil {
		return err
	}

	// 通过WebSocket通知会话的所有成员有新成员加入
	if websocket.GlobalHub != nil {
		// 获取会话的所有成员
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			// 提取成员ID列表
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, member := range members {
				memberIDs = append(memberIDs, member.UserID)
			}

			// 通知所有成员
			for _, memberID := range memberIDs {
				websocket.GlobalHub.SendToUser(memberID, "conversation_member_added", map[string]interface{}{
					"conversation_id": conversationIDStr,
					"user_id":         targetUserID,
					"role":            role,
					"added_by":        userID,
				})
			}
			logger.InfofWithCaller("Member added notification sent to %d members", len(memberIDs))
		}
	}

	return nil
}

// RemoveMemberFromConversation 从会话中移除成员
func (s *MemberService) RemoveMemberFromConversation(ctx context.Context, conversationIDStr, userID, targetUserID string) error {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	targetUUID, err := uuid.Parse(targetUserID)
	if err != nil {
		return err
	}

	// 检查操作者是否是会话的管理员或拥有者
	enrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, userUUID)
	if err != nil {
		return errors.New("not authorized")
	}

	if enrollment.Role != models.EnrollmentRoleOwner && enrollment.Role != models.EnrollmentRoleAdmin {
		return errors.New("not authorized")
	}

	// 不能移除拥有者
	targetEnrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, targetUUID)
	if err != nil {
		return errors.New("user not in conversation")
	}

	if targetEnrollment.Role == models.EnrollmentRoleOwner {
		return errors.New("cannot remove owner")
	}

	// 移除成员
	err = s.enrollmentRepo.DeleteByConversationAndUser(ctx, conversationID, targetUUID)
	if err != nil {
		return err
	}

	// 通过WebSocket通知会话的所有成员有成员被移除
	if websocket.GlobalHub != nil {
		// 获取会话的所有成员
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			// 提取成员ID列表
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, member := range members {
				memberIDs = append(memberIDs, member.UserID)
			}

			// 通知所有成员
			for _, memberID := range memberIDs {
				websocket.GlobalHub.SendToUser(memberID, "conversation_member_removed", map[string]interface{}{
					"conversation_id": conversationIDStr,
					"user_id":         targetUserID,
					"removed_by":      userID,
				})
			}
			logger.InfofWithCaller("Member removed notification sent to %d members", len(memberIDs))
		}
	}

	return nil
}

// UpdateMemberRole 更新成员角色（转让群主/设置管理员）
func (s *MemberService) UpdateMemberRole(ctx context.Context, conversationIDStr, userID string, req *models.UpdateMemberRoleRequest) error {
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	// 验证操作者权限
	operatorEnrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, userUUID)
	if err != nil {
		return errors.New("not authorized")
	}

	// 只有 owner 可以转让群主或设置/撤销管理员
	if operatorEnrollment.Role != models.EnrollmentRoleOwner {
		return errors.New("only owner can update member roles")
	}

	// 查找目标成员
	targetEnrollment, err := s.enrollmentRepo.FindByConversationAndUser(ctx, conversationID, req.UserID)
	if err != nil {
		return errors.New("user not in conversation")
	}

	// 不能修改自己的角色
	if req.UserID == userUUID {
		return errors.New("cannot update your own role")
	}

	// 禁止修改 Bot 的角色（Bot 只能是 member）
	targetUser, _ := s.userRepo.FindByID(ctx, req.UserID)
	if targetUser != nil && targetUser.IsBot {
		if models.EnrollmentRole(req.Role) != models.EnrollmentRoleMember {
			return errors.New("bot can only be member")
		}
		return errors.New("bot can only be member")
	}

	// 如果操作者要转让群主
	if models.EnrollmentRole(req.Role) == models.EnrollmentRoleOwner {
		// 操作者降为 member
		operatorEnrollment.Role = models.EnrollmentRoleMember
		err = s.enrollmentRepo.Update(ctx, operatorEnrollment)
		if err != nil {
			return err
		}
		// 目标成员升级为 owner
		targetEnrollment.Role = models.EnrollmentRoleOwner
		err = s.enrollmentRepo.Update(ctx, targetEnrollment)
		if err != nil {
			return err
		}

		// 通过 WebSocket 通知所有成员
		if websocket.GlobalHub != nil {
			members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
			if err == nil {
				memberIDs := make([]uuid.UUID, 0, len(members))
				for _, m := range members {
					memberIDs = append(memberIDs, m.UserID)
				}
				for _, mID := range memberIDs {
					websocket.GlobalHub.SendToUser(mID, "conversation_member_role_updated", map[string]interface{}{
						"conversation_id": conversationIDStr,
						"user_id":         req.UserID.String(),
						"role":            req.Role,
						"updated_by":      userID,
					})
				}
			}
		}

		logger.InfofWithCaller("Ownership transferred from %s to %s in conversation %s", userID, req.UserID.String(), conversationIDStr)
		return nil
	}

	// 设置/撤销管理员
	targetEnrollment.Role = models.EnrollmentRole(req.Role)
	err = s.enrollmentRepo.Update(ctx, targetEnrollment)
	if err != nil {
		return err
	}

	// 通过 WebSocket 通知所有成员
	if websocket.GlobalHub != nil {
		members, err := s.enrollmentRepo.FindByConversationID(ctx, conversationID)
		if err == nil {
			memberIDs := make([]uuid.UUID, 0, len(members))
			for _, m := range members {
				memberIDs = append(memberIDs, m.UserID)
			}
			for _, mID := range memberIDs {
				websocket.GlobalHub.SendToUser(mID, "conversation_member_role_updated", map[string]interface{}{
					"conversation_id": conversationIDStr,
					"user_id":         req.UserID.String(),
					"role":            req.Role,
					"updated_by":      userID,
				})
			}
		}
	}

	logger.InfofWithCaller("Member %s role updated to %s in conversation %s by %s", req.UserID.String(), req.Role, conversationIDStr, userID)
	return nil
}
