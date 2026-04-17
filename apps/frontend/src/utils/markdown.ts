import { marked } from 'marked';
import DOMPurify from 'dompurify';

// 配置 marked：启用 GFM（GitHub Flavored Markdown）
marked.setOptions({
  gfm: true,
  breaks: true,
});

// 配置 DOMPurify：允许代码相关标签和属性
const PURIFY_CONFIG: DOMPurify.Config = {
  ALLOWED_TAGS: [
    'h1',
    'h2',
    'h3',
    'h4',
    'h5',
    'h6',
    'p',
    'br',
    'hr',
    'ul',
    'ol',
    'li',
    'blockquote',
    'pre',
    'code',
    'a',
    'strong',
    'b',
    'em',
    'i',
    'del',
    's',
    'table',
    'thead',
    'tbody',
    'tr',
    'th',
    'td',
    'span',
    'div',
    'img',
  ],
  ALLOWED_ATTR: ['href', 'target', 'rel', 'src', 'alt', 'class'],
};

/**
 * 将 Markdown 文本安全地渲染为 HTML
 * 使用 marked 解析，DOMPurify 清理防止 XSS
 */
export function renderMarkdown(text: string): string {
  if (!text) return '';
  const rawHtml = marked.parse(text) as string;
  return DOMPurify.sanitize(rawHtml, PURIFY_CONFIG);
}
