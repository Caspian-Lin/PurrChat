import type { Message, SystemMessageContent, FileMessageContent } from '../models/types';

export function formatSystemMessageText(message: Message, currentUserId?: string): string {
  try {
    const sys = JSON.parse(message.content) as SystemMessageContent;
    switch (sys.type) {
      case 'workflow_start':
        return `${sys.bot_name || 'Bot'} 进入了 Agent 模式`;
      case 'workflow_end':
        return `${sys.bot_name || 'Bot'} 退出了 Agent 模式`;
      case 'bot_deployed':
        return `${sys.bot_name || 'Bot'} 已加入对话`;
      case 'bot_undeployed':
        return `${sys.bot_name || 'Bot'} 已离开对话`;
      case 'poke': {
        const pokerName = message.sender?.username || '某人';
        const isSelfPoker = message.sender_id === currentUserId;
        const isSelfTarget = sys.user_id === currentUserId;

        if (isSelfPoker && isSelfTarget) return '你 拍了拍 自己';
        if (isSelfPoker) return `你 拍了拍 ${sys.user_name}`;
        if (isSelfTarget) return `${pokerName} 拍了拍 你`;
        return `${pokerName} 拍了拍 ${sys.user_name}`;
      }
      default:
        return message.content;
    }
  } catch {
    return message.content;
  }
}

function tryParseFileContent(content: string): FileMessageContent | null {
  try {
    return JSON.parse(content) as FileMessageContent;
  } catch {
    return null;
  }
}

export function formatLastMessagePreview(
  message: Message | undefined,
  currentUserId?: string,
  isGroup?: boolean
): string {
  if (!message) return '暂无消息';
  if (message.msg_type === 'image') return '[图片]';
  if (message.msg_type === 'file') {
    const fileContent = tryParseFileContent(message.content);
    if (fileContent?.thumbnail_url) return '[图片]';
    return fileContent ? `[文件] ${fileContent.file_name}` : '[文件]';
  }
  if (message.msg_type === 'system') {
    return formatSystemMessageText(message, currentUserId);
  }

  const content = message.content || '暂无消息';
  const senderName =
    isGroup && message.sender_id !== currentUserId
      ? message.sender?.username || message.bot_name
      : undefined;
  return senderName ? `${senderName}: ${content}` : content;
}
