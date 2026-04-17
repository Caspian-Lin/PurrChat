import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

// vi.mock factory — must not reference variables outside the factory
vi.mock('../services/websocket', () => ({
  websocketService: {
    on: vi.fn(),
    off: vi.fn(),
    emit: vi.fn(),
    connect: vi.fn(),
    disconnect: vi.fn(),
  },
}));

vi.mock('../services/messageCache', () => ({
  useMessageCache: () => ({
    addMessage: vi.fn(),
    addMessages: vi.fn(),
    getMessages: vi.fn().mockReturnValue([]),
    hasCache: vi.fn().mockReturnValue(false),
    getLastUpdated: vi.fn().mockReturnValue(0),
    init: vi.fn(),
    clearAll: vi.fn(),
  }),
}));

import { websocketService } from '../services/websocket';
import { websocketEventManager } from '../services/websocketEventManager';

// At this point, the WebSocketEventManager constructor has already run,
// registering handlers via websocketService.on. Capture the handler map.
const registeredHandlers = new Map<string, Function[]>();

// Replay the on() calls to capture handlers
function captureHandlers() {
  const calls = (websocketService.on as ReturnType<typeof vi.fn>).mock.calls;
  registeredHandlers.clear();
  for (const call of calls) {
    const event = call[0] as string;
    const handler = call[1] as Function;
    if (!registeredHandlers.has(event)) {
      registeredHandlers.set(event, []);
    }
    registeredHandlers.get(event)!.push(handler);
  }
}

// Capture immediately after module import
captureHandlers();

// Helper to trigger an event through all registered handlers
function triggerEvent(event: string, data: any) {
  const handlers = registeredHandlers.get(event);
  if (handlers) {
    for (const handler of handlers) {
      handler(data);
    }
  }
}

describe('WebSocketEventManager', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    // Note: we do NOT clearAllMocks here because that would destroy
    // the websocketService.on call records captured during module init.
    // Instead, clear spy call counts but preserve the mock.calls array.
    (websocketService.on as ReturnType<typeof vi.fn>).mockClear();
    (websocketService.off as ReturnType<typeof vi.fn>).mockClear();
  });

  describe('Callback registration', () => {
    it('onConversationUpdate / offConversationUpdate should add/remove callbacks', () => {
      const cb = vi.fn();
      websocketEventManager.onConversationUpdate(cb);

      triggerEvent('new_group_conversation', {
        conversation_id: 'conv-1', name: 'Test', created_by: 'user-1', member_count: 2,
      });
      expect(cb).toHaveBeenCalled();

      cb.mockClear();
      websocketEventManager.offConversationUpdate(cb);

      triggerEvent('new_group_conversation', {
        conversation_id: 'conv-1', name: 'Test', created_by: 'user-1', member_count: 2,
      });
      expect(cb).not.toHaveBeenCalled();
    });

    it('onMessageUpdate / offMessageUpdate should add/remove callbacks', () => {
      const cb = vi.fn();
      websocketEventManager.onMessageUpdate(cb);
      websocketEventManager.setCurrentConversation('conv-1');

      triggerEvent('new_message', {
        id: 'msg-1', conversation_id: 'conv-1', sender_id: 'user-2',
        content: 'Hi', msg_type: 'text', created_at: new Date().toISOString(),
      });
      expect(cb).toHaveBeenCalled();

      cb.mockClear();
      websocketEventManager.offMessageUpdate(cb);

      triggerEvent('new_message', {
        id: 'msg-2', conversation_id: 'conv-1', sender_id: 'user-2',
        content: 'Hi again', msg_type: 'text', created_at: new Date().toISOString(),
      });
      expect(cb).not.toHaveBeenCalled();
    });

    it('onFriendRequest / offFriendRequest should add/remove callbacks', () => {
      const cb = vi.fn();
      websocketEventManager.onFriendRequest(cb);

      triggerEvent('new_friend_request', {
        conversation_id: 'conv-1', sender_id: 'user-2', status: 'pending',
        sender: { id: 'user-2', username: 'sender', avatar_url: '' },
      });
      expect(cb).toHaveBeenCalled();

      cb.mockClear();
      websocketEventManager.offFriendRequest(cb);

      triggerEvent('new_friend_request', {
        conversation_id: 'conv-1', sender_id: 'user-2', status: 'pending',
        sender: { id: 'user-2', username: 'sender', avatar_url: '' },
      });
      expect(cb).not.toHaveBeenCalled();
    });

    it('onOnlineStatus / offOnlineStatus should add/remove callbacks', () => {
      const cb = vi.fn();
      websocketEventManager.onOnlineStatus(cb);

      triggerEvent('user_online_status', { user_id: 'user-2', online: true });
      expect(cb).toHaveBeenCalledWith('user-2', true);

      cb.mockClear();
      websocketEventManager.offOnlineStatus(cb);

      triggerEvent('user_online_status', { user_id: 'user-2', online: false });
      expect(cb).not.toHaveBeenCalled();
    });
  });

  describe('getUserOnlineStatus', () => {
    it('should return false for unknown user', () => {
      expect(websocketEventManager.getUserOnlineStatus('unknown')).toBe(false);
    });

    it('should return cached online status', () => {
      triggerEvent('user_online_status', { user_id: 'user-2', online: true });
      expect(websocketEventManager.getUserOnlineStatus('user-2')).toBe(true);
    });
  });

  describe('handleNewMessage', () => {
    it('should trigger conversationUpdateCallbacks', () => {
      const convCb = vi.fn();
      websocketEventManager.onConversationUpdate(convCb);

      triggerEvent('new_message', {
        id: 'msg-1', conversation_id: 'conv-1', sender_id: 'user-2',
        content: 'Hi', msg_type: 'text', created_at: new Date().toISOString(),
      });

      expect(convCb).toHaveBeenCalledWith(
        expect.objectContaining({ id: 'conv-1' })
      );
    });

    it('should not trigger messageUpdateCallbacks for non-current conversation', () => {
      const msgCb = vi.fn();
      websocketEventManager.onMessageUpdate(msgCb);
      websocketEventManager.setCurrentConversation('conv-other');

      triggerEvent('new_message', {
        id: 'msg-1', conversation_id: 'conv-1', sender_id: 'user-2',
        content: 'Hi', msg_type: 'text', created_at: new Date().toISOString(),
      });

      expect(msgCb).not.toHaveBeenCalled();
    });

    it('should trigger messageUpdateCallbacks for current conversation', () => {
      const msgCb = vi.fn();
      websocketEventManager.onMessageUpdate(msgCb);
      websocketEventManager.setCurrentConversation('conv-1');

      triggerEvent('new_message', {
        id: 'msg-1', conversation_id: 'conv-1', sender_id: 'user-2',
        content: 'Hi', msg_type: 'text', created_at: new Date().toISOString(),
      });

      expect(msgCb).toHaveBeenCalledWith('conv-1', expect.objectContaining({ id: 'msg-1' }));
    });
  });

  describe('handleNewFriendRequest', () => {
    it('should trigger friendRequestCallbacks when sender exists', () => {
      const friendCb = vi.fn();
      websocketEventManager.onFriendRequest(friendCb);

      triggerEvent('new_friend_request', {
        conversation_id: 'conv-1', sender_id: 'user-2', status: 'pending',
        sender: { id: 'user-2', username: 'sender', avatar_url: '' },
      });

      expect(friendCb).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'pending' })
      );
    });

    it('should trigger conversationUpdateCallbacks', () => {
      const convCb = vi.fn();
      websocketEventManager.onConversationUpdate(convCb);

      triggerEvent('new_friend_request', {
        conversation_id: 'conv-1', sender_id: 'user-2', status: 'pending',
        sender: { id: 'user-2', username: 'sender', avatar_url: '' },
      });

      expect(convCb).toHaveBeenCalled();
    });

    it('should not trigger friendRequestCallbacks when sender is missing', () => {
      const friendCb = vi.fn();
      websocketEventManager.onFriendRequest(friendCb);

      triggerEvent('new_friend_request', {
        conversation_id: 'conv-1', sender_id: 'user-2', status: 'pending',
      });

      expect(friendCb).not.toHaveBeenCalled();
    });
  });

  describe('handleFriendRequestUpdate', () => {
    it('should trigger friendRequestCallbacks with correct status', () => {
      const friendCb = vi.fn();
      websocketEventManager.onFriendRequest(friendCb);

      triggerEvent('friend_request_update', {
        conversation_id: 'conv-1', status: 'accepted', action: 'accept', user_id: 'user-2',
      });

      expect(friendCb).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'accepted' })
      );
    });

    it('should trigger conversationUpdateCallbacks', () => {
      const convCb = vi.fn();
      websocketEventManager.onConversationUpdate(convCb);

      triggerEvent('friend_request_update', {
        conversation_id: 'conv-1', status: 'accepted', action: 'accept', user_id: 'user-2',
      });

      expect(convCb).toHaveBeenCalled();
    });
  });

  describe('handleConversationMemberRemoved', () => {
    it('should trigger conversationUpdateCallbacks', () => {
      const convCb = vi.fn();
      websocketEventManager.onConversationUpdate(convCb);

      triggerEvent('conversation_member_removed', {
        conversation_id: 'conv-1', user_id: 'user-other', removed_by: 'user-owner',
      });

      expect(convCb).toHaveBeenCalled();
    });
  });

  describe('handleUserOnlineStatus', () => {
    it('should update onlineStatusCache and trigger callbacks', () => {
      const statusCb = vi.fn();
      websocketEventManager.onOnlineStatus(statusCb);

      triggerEvent('user_online_status', { user_id: 'user-2', online: true });
      expect(websocketEventManager.getUserOnlineStatus('user-2')).toBe(true);
      expect(statusCb).toHaveBeenCalledWith('user-2', true);

      triggerEvent('user_online_status', { user_id: 'user-2', online: false });
      expect(websocketEventManager.getUserOnlineStatus('user-2')).toBe(false);
    });
  });

  describe('destroy', () => {
    it('should clear all callbacks so events no longer trigger them', () => {
      const cb = vi.fn();
      websocketEventManager.onConversationUpdate(cb);

      triggerEvent('new_group_conversation', {
        conversation_id: 'conv-1', name: 'Test', created_by: 'user-1', member_count: 2,
      });
      expect(cb).toHaveBeenCalledTimes(1);

      websocketEventManager.destroy();

      triggerEvent('new_group_conversation', {
        conversation_id: 'conv-1', name: 'Test', created_by: 'user-1', member_count: 2,
      });
      expect(cb).toHaveBeenCalledTimes(1); // Still 1, not called again
    });

    it('should clear onlineStatusCache', () => {
      triggerEvent('user_online_status', { user_id: 'user-2', online: true });
      expect(websocketEventManager.getUserOnlineStatus('user-2')).toBe(true);

      websocketEventManager.destroy();

      expect(websocketEventManager.getUserOnlineStatus('user-2')).toBe(false);
    });
  });
});
