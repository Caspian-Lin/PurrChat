import { useRouter } from 'vue-router';

// Panel控制器
export function usePanelController() {
  const router = useRouter();

  // 导航到指定的panel
  const navigateToPanel = (panel: 'chat' | 'friends' | 'ai' | 'bots' | 'settings') => {
    const routes: Record<string, string> = {
      chat: '/chat',
      friends: '/friends',
      ai: '/ai',
      bots: '/bots',
      settings: '/settings',
    };
    router.push(routes[panel]!);
  };

  return {
    navigateToPanel,
  };
}
