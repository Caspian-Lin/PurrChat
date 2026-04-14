import { useRouter } from 'vue-router';

// Panel控制器
export function usePanelController() {
  const router = useRouter();

  // 导航到指定的panel
  const navigateToPanel = (panel: 'chat' | 'friends' | 'ai') => {
    if (panel === 'chat') {
      router.push('/chat');
    } else if (panel === 'friends') {
      router.push('/friends');
    } else if (panel === 'ai') {
      router.push('/ai');
    }
  };

  return {
    navigateToPanel,
  };
}
