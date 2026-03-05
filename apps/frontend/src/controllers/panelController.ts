import { useRouter } from 'vue-router';

// Panel控制器
export function usePanelController() {
  const router = useRouter();

  // 导航到指定的panel
  const navigateToPanel = (panel: 'chat' | 'friends') => {
    if (panel === 'chat') {
      router.push('/chat');
    } else if (panel === 'friends') {
      router.push('/friends');
    }
  };

  return {
    navigateToPanel,
  };
}
