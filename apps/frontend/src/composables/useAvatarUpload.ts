import { ref } from 'vue';
import { api } from '../models/api';
import { useAuthStore } from '../stores/auth';
import { threeStageUpload } from '../utils/upload';

const MAX_AVATAR_SIZE = 2 * 1024 * 1024; // 2MB
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/bmp'];

export function useAvatarUpload() {
  const uploading = ref(false);
  const error = ref<string | null>(null);
  const previewUrl = ref<string | null>(null);

  const authStore = useAuthStore();

  // 校验文件
  function validateFile(file: File): string | null {
    if (!ALLOWED_TYPES.includes(file.type)) {
      return '不支持的图片格式，请使用 JPG、PNG、GIF、WebP 或 BMP';
    }
    if (file.size > MAX_AVATAR_SIZE) {
      return '图片大小不能超过 2MB';
    }
    return null;
  }

  // 上传头像
  async function uploadAvatar(file: File): Promise<string | null> {
    const validationError = validateFile(file);
    if (validationError) {
      error.value = validationError;
      return null;
    }

    uploading.value = true;
    error.value = null;

    // 生成本地预览
    previewUrl.value = URL.createObjectURL(file);

    try {
      // 三阶段上传（申请 → PUT → 确认）
      const { publicUrl } = await threeStageUpload(file, 'avatar', 'avatar');
      console.log('[avatar-upload] 存储服务返回 public_url:', publicUrl);

      // 第四步：更新用户资料中的头像 URL
      const profileResp = await api.updateProfile({ avatar_url: publicUrl });

      if (!profileResp.success || !profileResp.data) {
        throw new Error(profileResp.message || '更新头像失败');
      }

      // 更新 auth store 中的用户数据
      authStore.user = profileResp.data;
      localStorage.setItem('user', JSON.stringify(profileResp.data));
      console.log('[avatar-upload] 头像更新成功，avatar_url:', profileResp.data.avatar_url);

      // 释放本地预览 URL
      if (previewUrl.value) {
        URL.revokeObjectURL(previewUrl.value);
      }
      previewUrl.value = null;

      return publicUrl;
    } catch (err: any) {
      error.value = err.message || '上传失败，请重试';
      // 释放本地预览 URL
      if (previewUrl.value) {
        URL.revokeObjectURL(previewUrl.value);
      }
      previewUrl.value = null;
      return null;
    } finally {
      uploading.value = false;
    }
  }

  // 清除错误
  function clearError() {
    error.value = null;
  }

  return {
    uploading,
    error,
    previewUrl,
    uploadAvatar,
    clearError,
  };
}
