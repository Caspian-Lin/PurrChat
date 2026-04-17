import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { useUserSearch } from '../composables/useUserSearch';
import type { User } from '../models/types';

// Mock api module
vi.mock('../models/api', () => ({
  api: {
    searchUsers: vi.fn(),
  },
}));

describe('useUserSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const mockUser: User = {
    id: 'user-1',
    uid: 1001,
    username: 'testuser',
    avatar_url: '',
    email: 'test@example.com',
    email_verified: true,
    phone: '13800138001',
    phone_verified: true,
    created_at: '2024-01-01T00:00:00Z',
  };

  describe('searchUsers', () => {
    it('should not search when query is empty', async () => {
      const { api } = await import('../models/api');
      const { searchUsers, searchQuery } = useUserSearch();

      searchQuery.value = '   ';
      await searchUsers();

      expect(api.searchUsers).not.toHaveBeenCalled();
    });

    it('should set searchResults on success', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.searchUsers).mockResolvedValueOnce({
        success: true,
        data: [mockUser],
      });

      const { searchUsers, searchResults, showSearchResults, searchQuery } = useUserSearch();
      searchQuery.value = 'test';
      await searchUsers();

      expect(searchResults.value).toHaveLength(1);
      expect(showSearchResults.value).toBe(true);
    });

    it('should handle API error gracefully', async () => {
      const { api } = await import('../models/api');
      vi.mocked(api.searchUsers).mockRejectedValueOnce(new Error('Network error'));

      const { searchUsers, searchResults, searchQuery } = useUserSearch();
      searchQuery.value = 'test';
      await searchUsers();

      expect(searchResults.value).toHaveLength(0);
    });
  });

  describe('groupedResults', () => {
    it('should group results by uid match', () => {
      const { searchQuery, searchResults, groupedResults } = useUserSearch();
      searchQuery.value = '1001';
      searchResults.value = [mockUser];

      expect(groupedResults.value.uidMatches).toHaveLength(1);
      expect(groupedResults.value.emailMatches).toHaveLength(0);
      expect(groupedResults.value.phoneMatches).toHaveLength(0);
    });

    it('should group results by email match', () => {
      const { searchQuery, searchResults, groupedResults } = useUserSearch();
      searchQuery.value = 'test@example';
      searchResults.value = [mockUser];

      expect(groupedResults.value.uidMatches).toHaveLength(0);
      expect(groupedResults.value.emailMatches).toHaveLength(1);
    });

    it('should group results by phone match', () => {
      const { searchQuery, searchResults, groupedResults } = useUserSearch();
      searchQuery.value = '138';
      searchResults.value = [mockUser];

      expect(groupedResults.value.phoneMatches).toHaveLength(1);
    });

    it('should return empty groups when no results', () => {
      const { searchQuery, searchResults, groupedResults } = useUserSearch();
      searchQuery.value = 'nothing';
      searchResults.value = [];

      expect(groupedResults.value.uidMatches).toHaveLength(0);
      expect(groupedResults.value.emailMatches).toHaveLength(0);
      expect(groupedResults.value.phoneMatches).toHaveLength(0);
    });
  });

  describe('highlightMatch', () => {
    it('should wrap matching text in span with highlight classes', () => {
      const { highlightMatch } = useUserSearch();

      const result = highlightMatch('testuser', 'test');

      expect(result).toContain('test');
      expect(result).toContain('class=');
    });

    it('should handle empty text', () => {
      const { highlightMatch } = useUserSearch();
      expect(highlightMatch('', 'test')).toBe('');
    });

    it('should handle empty query', () => {
      const { highlightMatch } = useUserSearch();
      expect(highlightMatch('testuser', '')).toBe('testuser');
    });

    it('should be case-insensitive', () => {
      const { highlightMatch } = useUserSearch();

      const result = highlightMatch('TestUser', 'test');

      expect(result).toContain('class=');
    });
  });

  describe('handleSearchBlur', () => {
    it('should set showSearchResults=false after delay', async () => {
      const { showSearchResults, handleSearchBlur, searchQuery } = useUserSearch();
      searchQuery.value = 'test';
      showSearchResults.value = true;

      handleSearchBlur();

      // Before delay
      expect(showSearchResults.value).toBe(true);

      // After 200ms delay
      vi.advanceTimersByTime(200);
      expect(showSearchResults.value).toBe(false);
    });
  });

  describe('clearSearchResults', () => {
    it('should clear searchQuery, searchResults, showSearchResults', () => {
      const { searchQuery, searchResults, showSearchResults, clearSearchResults } = useUserSearch();
      searchQuery.value = 'test';
      searchResults.value = [mockUser];
      showSearchResults.value = true;

      clearSearchResults();

      expect(searchQuery.value).toBe('');
      expect(searchResults.value).toHaveLength(0);
      expect(showSearchResults.value).toBe(false);
    });
  });
});
