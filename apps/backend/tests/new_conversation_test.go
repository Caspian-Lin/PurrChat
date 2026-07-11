package tests

import (
	"context"
	"testing"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
)

// TestNewConversation 测试新的会话功能
func TestNewConversation(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)

	ctx := context.Background()

	// 初始化repositories
	userRepo := repository.NewUserRepository()
	conversationRepo := repository.NewConversationRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	conversationMessageRepo := repository.NewConversationMessageRepository()
	friendshipRepo := repository.NewFriendshipRepository()

	// 创建service
	conversationService := services.NewConversationService(
		userRepo,
		conversationRepo,
		enrollmentRepo,
		conversationMessageRepo,
		friendshipRepo,
	)
	messageService := services.NewMessageService(
		userRepo,
		conversationRepo,
		enrollmentRepo,
		conversationMessageRepo,
		nil,
		nil,
	)
	memberService := services.NewMemberService(
		userRepo,
		conversationRepo,
		enrollmentRepo,
	)

	// 测试1: 创建私聊会话
	t.Run("CreateDirectConversation", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建两个测试用户
		user1, err := createTestUser(ctx, userRepo, "user1_direct")
		if err != nil {
			t.Fatalf("Failed to create user1: %v", err)
		}

		user2, err := createTestUser(ctx, userRepo, "user2_direct")
		if err != nil {
			t.Fatalf("Failed to create user2: %v", err)
		}

		// 创建私聊会话
		conversation, err := conversationService.CreateConversation(ctx, user1.ID.String(), user2.ID.String())
		if err != nil {
			t.Fatalf("Failed to create direct conversation: %v", err)
		}

		// 验证会话类型
		if conversation.ConversationType != models.ConversationTypeDirect {
			t.Errorf("Expected conversation type 'direct', got '%s'", conversation.ConversationType)
		}

		// 验证创建者
		if conversation.CreatedBy == nil || *conversation.CreatedBy != user1.ID {
			t.Errorf("Expected creator to be user1")
		}

		// 验证enrollment记录
		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get enrollments: %v", err)
		}

		if len(enrollments) != 2 {
			t.Errorf("Expected 2 enrollments, got %d", len(enrollments))
		}

		// 验证消息表已创建
		_, err = conversationMessageRepo.FindMessages(ctx, conversation.ID, 10, 0)
		if err != nil {
			t.Errorf("Failed to query messages from new table: %v", err)
		}
	})

	// 测试2: 创建群聊会话
	t.Run("CreateGroupConversation", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建三个测试用户
		owner, err := createTestUser(ctx, userRepo, "owner_group")
		if err != nil {
			t.Fatalf("Failed to create owner: %v", err)
		}

		member1, err := createTestUser(ctx, userRepo, "member1_group")
		if err != nil {
			t.Fatalf("Failed to create member1: %v", err)
		}

		member2, err := createTestUser(ctx, userRepo, "member2_group")
		if err != nil {
			t.Fatalf("Failed to create member2: %v", err)
		}

		// 创建群聊会话
		conversation, err := conversationService.CreateGroupConversation(
			ctx,
			owner.ID.String(),
			"测试群聊",
			[]string{member1.ID.String(), member2.ID.String()},
		)
		if err != nil {
			t.Fatalf("Failed to create group conversation: %v", err)
		}

		// 验证会话类型
		if conversation.ConversationType != models.ConversationTypeGroup {
			t.Errorf("Expected conversation type 'group', got '%s'", conversation.ConversationType)
		}

		// 验证会话名称
		if conversation.Name != "测试群聊" {
			t.Errorf("Expected conversation name '测试群聊', got '%s'", conversation.Name)
		}

		// 验证创建者
		if conversation.CreatedBy == nil || *conversation.CreatedBy != owner.ID {
			t.Errorf("Expected creator to be owner")
		}

		// 验证enrollment记录（应该有3个成员）
		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get enrollments: %v", err)
		}

		if len(enrollments) != 3 {
			t.Errorf("Expected 3 enrollments, got %d", len(enrollments))
		}

		// 验证owner的角色
		var ownerEnrollment *models.Enrollment
		for _, e := range enrollments {
			if e.UserID == owner.ID {
				ownerEnrollment = e
				break
			}
		}

		if ownerEnrollment == nil {
			t.Fatal("Owner enrollment not found")
		}

		if ownerEnrollment.Role != models.EnrollmentRoleOwner {
			t.Errorf("Expected owner role 'owner', got '%s'", ownerEnrollment.Role)
		}

		// 验证成员的角色
		var member1Enrollment *models.Enrollment
		for _, e := range enrollments {
			if e.UserID == member1.ID {
				member1Enrollment = e
				break
			}
		}

		if member1Enrollment == nil {
			t.Fatal("Member1 enrollment not found")
		}

		if member1Enrollment.Role != models.EnrollmentRoleMember {
			t.Errorf("Expected member1 role 'member', got '%s'", member1Enrollment.Role)
		}

		// 验证消息表已创建
		_, err = conversationMessageRepo.FindMessages(ctx, conversation.ID, 10, 0)
		if err != nil {
			t.Errorf("Failed to query messages from new table: %v", err)
		}
	})

	// 测试3: 添加成员到群聊
	t.Run("AddMemberToConversation", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建测试用户
		owner, err := createTestUser(ctx, userRepo, "owner_add")
		if err != nil {
			t.Fatalf("Failed to create owner: %v", err)
		}

		member1, err := createTestUser(ctx, userRepo, "member1_add")
		if err != nil {
			t.Fatalf("Failed to create member1: %v", err)
		}

		member2, err := createTestUser(ctx, userRepo, "member2_add")
		if err != nil {
			t.Fatalf("Failed to create member2: %v", err)
		}

		// 创建群聊会话
		conversation, err := conversationService.CreateGroupConversation(
			ctx,
			owner.ID.String(),
			"测试群聊添加成员",
			[]string{member1.ID.String()},
		)
		if err != nil {
			t.Fatalf("Failed to create group conversation: %v", err)
		}

		// 添加新成员
		err = memberService.AddMemberToConversation(
			ctx,
			conversation.ID.String(),
			owner.ID.String(),
			member2.ID.String(),
			models.EnrollmentRoleMember,
		)
		if err != nil {
			t.Fatalf("Failed to add member: %v", err)
		}

		// 验证成员数量
		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get enrollments: %v", err)
		}

		if len(enrollments) != 3 {
			t.Errorf("Expected 3 enrollments after adding member, got %d", len(enrollments))
		}
	})

	// 测试4: 从群聊移除成员
	t.Run("RemoveMemberFromConversation", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建测试用户
		owner, err := createTestUser(ctx, userRepo, "owner_remove")
		if err != nil {
			t.Fatalf("Failed to create owner: %v", err)
		}

		member1, err := createTestUser(ctx, userRepo, "member1_remove")
		if err != nil {
			t.Fatalf("Failed to create member1: %v", err)
		}

		member2, err := createTestUser(ctx, userRepo, "member2_remove")
		if err != nil {
			t.Fatalf("Failed to create member2: %v", err)
		}

		// 创建群聊会话
		conversation, err := conversationService.CreateGroupConversation(
			ctx,
			owner.ID.String(),
			"测试群聊移除成员",
			[]string{member1.ID.String(), member2.ID.String()},
		)
		if err != nil {
			t.Fatalf("Failed to create group conversation: %v", err)
		}

		// 移除成员
		err = memberService.RemoveMemberFromConversation(
			ctx,
			conversation.ID.String(),
			owner.ID.String(),
			member2.ID.String(),
		)
		if err != nil {
			t.Fatalf("Failed to remove member: %v", err)
		}

		// 验证成员数量
		enrollments, err := enrollmentRepo.FindByConversationID(ctx, conversation.ID)
		if err != nil {
			t.Fatalf("Failed to get enrollments: %v", err)
		}

		if len(enrollments) != 2 {
			t.Errorf("Expected 2 enrollments after removing member, got %d", len(enrollments))
		}

		// 验证被移除的成员不再存在
		_, err = enrollmentRepo.FindByConversationAndUser(ctx, conversation.ID, member2.ID)
		if err == nil {
			t.Error("Expected removed member to not exist")
		}
	})

	// 测试5: 获取会话成员
	t.Run("GetConversationMembers", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建测试用户
		owner, err := createTestUser(ctx, userRepo, "owner_get")
		if err != nil {
			t.Fatalf("Failed to create owner: %v", err)
		}

		member1, err := createTestUser(ctx, userRepo, "member1_get")
		if err != nil {
			t.Fatalf("Failed to create member1: %v", err)
		}

		member2, err := createTestUser(ctx, userRepo, "member2_get")
		if err != nil {
			t.Fatalf("Failed to create member2: %v", err)
		}

		// 创建群聊会话
		conversation, err := conversationService.CreateGroupConversation(
			ctx,
			owner.ID.String(),
			"测试群聊获取成员",
			[]string{member1.ID.String(), member2.ID.String()},
		)
		if err != nil {
			t.Fatalf("Failed to create group conversation: %v", err)
		}

		// 获取会话成员
		members, err := conversationService.GetConversationMembers(ctx, owner.ID.String(), conversation.ID.String())
		if err != nil {
			t.Fatalf("Failed to get conversation members: %v", err)
		}

		// 验证成员数量
		if len(members) != 3 {
			t.Errorf("Expected 3 members, got %d", len(members))
		}

		// 验证成员包含用户信息
		var foundOwner, foundMember1, foundMember2 bool
		for _, m := range members {
			if m.User != nil {
				if m.User.ID == owner.ID {
					foundOwner = true
				} else if m.User.ID == member1.ID {
					foundMember1 = true
				} else if m.User.ID == member2.ID {
					foundMember2 = true
				}
			}
		}

		if !foundOwner {
			t.Error("Owner not found in members list")
		}
		if !foundMember1 {
			t.Error("Member1 not found in members list")
		}
		if !foundMember2 {
			t.Error("Member2 not found in members list")
		}
	})

	// 测试6: 查找用户的会话
	t.Run("FindUserConversations", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建测试用户
		user1, err := createTestUser(ctx, userRepo, "user1_find")
		if err != nil {
			t.Fatalf("Failed to create user1: %v", err)
		}

		user2, err := createTestUser(ctx, userRepo, "user2_find")
		if err != nil {
			t.Fatalf("Failed to create user2: %v", err)
		}

		// 创建私聊会话
		conversation, err := conversationService.CreateConversation(ctx, user1.ID.String(), user2.ID.String())
		if err != nil {
			t.Fatalf("Failed to create conversation: %v", err)
		}

		// 查找user1的会话
		conversations, err := conversationService.GetConversations(ctx, user1.ID.String())
		if err != nil {
			t.Fatalf("Failed to get conversations: %v", err)
		}

		// 验证会话存在
		if len(conversations) == 0 {
			t.Error("Expected at least one conversation")
		}

		// 验证会话包含成员信息
		var foundConversation *models.Conversation
		for _, c := range conversations {
			if c.ID == conversation.ID {
				foundConversation = c
				break
			}
		}

		if foundConversation == nil {
			t.Fatal("Conversation not found in user's conversations list")
		}

		if foundConversation.Members == nil {
			t.Error("Expected conversation to have members")
		} else if len(foundConversation.Members) != 2 {
			t.Errorf("Expected 2 members, got %d", len(foundConversation.Members))
		}
	})

	// 测试7: 非管理员不能添加成员
	t.Run("NonAdminCannotAddMember", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建测试用户
		owner, err := createTestUser(ctx, userRepo, "owner_perm")
		if err != nil {
			t.Fatalf("Failed to create owner: %v", err)
		}

		member1, err := createTestUser(ctx, userRepo, "member1_perm")
		if err != nil {
			t.Fatalf("Failed to create member1: %v", err)
		}

		member2, err := createTestUser(ctx, userRepo, "member2_perm")
		if err != nil {
			t.Fatalf("Failed to create member2: %v", err)
		}

		// 创建群聊会话
		conversation, err := conversationService.CreateGroupConversation(
			ctx,
			owner.ID.String(),
			"测试群聊权限",
			[]string{member1.ID.String()},
		)
		if err != nil {
			t.Fatalf("Failed to create group conversation: %v", err)
		}

		// 尝试用普通成员添加新成员（应该失败）
		err = memberService.AddMemberToConversation(
			ctx,
			conversation.ID.String(),
			member1.ID.String(),
			member2.ID.String(),
			models.EnrollmentRoleMember,
		)
		if err == nil {
			t.Error("Expected error when non-admin tries to add member")
		}
	})

	// 测试8: 不能移除owner
	t.Run("CannotRemoveOwner", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建测试用户
		owner, err := createTestUser(ctx, userRepo, "owner_owner")
		if err != nil {
			t.Fatalf("Failed to create owner: %v", err)
		}

		member1, err := createTestUser(ctx, userRepo, "member1_owner")
		if err != nil {
			t.Fatalf("Failed to create member1: %v", err)
		}

		// 创建群聊会话
		conversation, err := conversationService.CreateGroupConversation(
			ctx,
			owner.ID.String(),
			"测试群聊移除owner",
			[]string{member1.ID.String()},
		)
		if err != nil {
			t.Fatalf("Failed to create group conversation: %v", err)
		}

		// 尝试移除owner（应该失败）
		err = memberService.RemoveMemberFromConversation(
			ctx,
			conversation.ID.String(),
			owner.ID.String(),
			owner.ID.String(),
		)
		if err == nil {
			t.Error("Expected error when trying to remove owner")
		}
	})

	// 测试9: 群聊消息发送
	t.Run("SendMessageToGroup", func(t *testing.T) {
		CleanupTestTables(t)
		// 创建测试用户
		owner, err := createTestUser(ctx, userRepo, "owner_msg")
		if err != nil {
			t.Fatalf("Failed to create owner: %v", err)
		}

		member1, err := createTestUser(ctx, userRepo, "member1_msg")
		if err != nil {
			t.Fatalf("Failed to create member1: %v", err)
		}

		// 创建群聊会话
		conversation, err := conversationService.CreateGroupConversation(
			ctx,
			owner.ID.String(),
			"测试群聊消息",
			[]string{member1.ID.String()},
		)
		if err != nil {
			t.Fatalf("Failed to create group conversation: %v", err)
		}

		// 发送消息
		sendReq := &models.SendMessageRequest{
			ConversationID: conversation.ID,
			Content:        "Hello, group!",
			MsgType:        "text",
		}
		message, err := messageService.SendMessage(ctx, owner.ID.String(), sendReq)
		if err != nil {
			t.Fatalf("Failed to send message: %v", err)
		}

		// 验证消息
		if message.Content != "Hello, group!" {
			t.Errorf("Expected message content 'Hello, group!', got '%s'", message.Content)
		}

		if message.SenderID != owner.ID {
			t.Error("Expected sender to be owner")
		}

		// 验证消息已保存
		messages, err := conversationMessageRepo.FindMessages(ctx, conversation.ID, 10, 0)
		if err != nil {
			t.Fatalf("Failed to get messages: %v", err)
		}

		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}
	})
}

// createTestUser 创建测试用户
func createTestUser(ctx context.Context, userRepo repository.UserRepository, username string) (*models.User, error) {
	phone := username
	if len(phone) > 10 {
		phone = phone[:10]
	}
	phone = "phone_" + phone

	user := &models.User{
		Username:      username,
		Email:         username + "@test.com",
		PasswordHash:  "test_hash",
		Salt:          "test_salt",
		UID:           1000 + len(username),
		Phone:         phone,
		EmailVerified: true,
		PhoneVerified: true,
	}

	err := userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
