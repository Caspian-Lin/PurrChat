import { ref } from 'vue';
import { storageApi, api } from '../models/api';
import { useAuthStore } from '../stores/auth';

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
      // 第一步：申请上传（获取预签名 URL）
      const requestResp = await storageApi.requestUpload({
        file_name: file.name,
        file_size: file.size,
        content_type: file.type,
        category: 'avatar',
        usage: 'avatar',
      });

      if (!requestResp.success || !requestResp.data) {
        throw new Error(requestResp.message || '申请上传失败');
      }

      const { upload_id, object_key, upload_url } = requestResp.data;

      // 第二步：使用预签名 URL 直接上传文件到对象存储
      const uploadResp = await fetch(upload_url, {
        method: 'PUT',
        body: file,
        headers: {
          'Content-Type': file.type,
        },
      });

      if (!uploadResp.ok) {
        throw new Error('文件上传失败');
      }

      // 第三步：确认上传
      const confirmResp = await storageApi.confirmUpload({
        upload_id,
        object_key,
      });

      if (!confirmResp.success || !confirmResp.data) {
        throw new Error(confirmResp.message || '确认上传失败');
      }

      const publicUrl = confirmResp.data.public_url;

      // 第四步：更新用户资料中的头像 URL
      const profileResp = await api.updateProfile({ avatar_url: publicUrl });

      if (!profileResp.success || !profileResp.data) {
        throw new Error(profileResp.message || '更新头像失败');
      }

      // 更新 auth store 中的用户数据
      authStore.user = profileResp.data;
      localStorage.setItem('user', JSON.stringify(profileResp.data));

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
