import { ref } from 'vue';

interface Message {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  content: string;
  duration?: number;
}

const messages = ref<Message[]>([]);

export const useMessage = () => {
  const addMessage = (type: Message['type'], content: string, duration = 3000) => {
    const id = Date.now().toString();
    messages.value.push({ id, type, content, duration });

    setTimeout(() => {
      removeMessage(id);
    }, duration);
  };

  const removeMessage = (id: string) => {
    const index = messages.value.findIndex((msg) => msg.id === id);
    if (index > -1) {
      messages.value.splice(index, 1);
    }
  };

  const success = (content: string, duration?: number) => {
    addMessage('success', content, duration);
  };

  const error = (content: string, duration?: number) => {
    addMessage('error', content, duration);
  };

  const warning = (content: string, duration?: number) => {
    addMessage('warning', content, duration);
  };

  const info = (content: string, duration?: number) => {
    addMessage('info', content, duration);
  };

  return {
    messages,
    success,
    error,
    warning,
    info,
  };
};
