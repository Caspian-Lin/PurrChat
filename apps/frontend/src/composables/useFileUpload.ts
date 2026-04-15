import { ref } from 'vue';
import { storageApi } from '../models/api';
import { useMessage } from './useMessage';
import type { FileMessageContent } from '../models/types';

const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50MB

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/bmp'];

export function useFileUpload() {
  const uploading = ref(false);
  const uploadProgress = ref(0);
  const error = ref<string | null>(null);
  const fileData = ref<FileMessageContent | null>(null);
  const thumbnailDataUrl = ref<string | null>(null);
  const message = useMessage();

  function isImageFile(file: File): boolean {
    return IMAGE_TYPES.includes(file.type);
  }

  function validateFile(file: File): string | null {
    if (file.size > MAX_FILE_SIZE) {
      return '文件大小不能超过 50MB';
    }
    return null;
  }

  // Canvas 生成缩略图 Blob（用于上传到存储服务）
  function generateThumbnail(file: File): Promise<Blob> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(file);
      img.onload = () => {
        const canvas = document.createElement('canvas');
        const maxWidth = 200;
        const scale = Math.min(maxWidth / img.width, 1);
        canvas.width = Math.round(img.width * scale);
        canvas.height = Math.round(img.height * scale);
        const ctx = canvas.getContext('2d')!;
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
        URL.revokeObjectURL(url);
        canvas.toBlob(
          (blob) => {
            if (blob) resolve(blob);
            else reject(new Error('生成缩略图失败'));
          },
          'image/jpeg',
          0.6
        );
      };
      img.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error('加载图片失败'));
      };
      img.src = url;
    });
  }

  // Canvas 生成本地预览 DataURL（上传前显示在输入框中）
  function generateLocalPreview(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(file);
      img.onload = () => {
        const canvas = document.createElement('canvas');
        const maxWidth = 200;
        const scale = Math.min(maxWidth / img.width, 1);
        canvas.width = Math.round(img.width * scale);
        canvas.height = Math.round(img.height * scale);
        const ctx = canvas.getContext('2d')!;
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
        URL.revokeObjectURL(url);
        resolve(canvas.toDataURL('image/jpeg', 0.6));
      };
      img.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error('生成预览失败'));
      };
      img.src = url;
    });
  }

  // 两阶段上传文件到存储服务
  async function uploadFile(
    file: File,
    category: 'chat-image' | 'file'
  ): Promise<{ fileId: string; publicUrl: string }> {
    const requestResp = await storageApi.requestUpload({
      file_name: file.name,
      file_size: file.size,
      content_type: file.type,
      category,
      usage: 'message',
    });
    if (!requestResp.success || !requestResp.data) {
      throw new Error(requestResp.message || '申请上传失败');
    }

    const { upload_id, object_key, upload_url } = requestResp.data;

    const uploadResp = await fetch(upload_url, {
      method: 'PUT',
      body: file,
      headers: { 'Content-Type': file.type },
    });
    if (!uploadResp.ok) {
      throw new Error('文件上传失败');
    }

    const confirmResp = await storageApi.confirmUpload({
      upload_id,
      object_key,
    });
    if (!confirmResp.success || !confirmResp.data) {
      throw new Error(confirmResp.message || '确认上传失败');
    }

    return {
      fileId: confirmResp.data.file_id,
      publicUrl: confirmResp.data.public_url,
    };
  }

  // 完整上传流程
  async function processAndUpload(file: File): Promise<FileMessageContent | null> {
    const validationError = validateFile(file);
    if (validationError) {
      error.value = validationError;
      message.error(validationError);
      return null;
    }

    uploading.value = true;
    uploadProgress.value = 0;
    error.value = null;
    fileData.value = null;
    thumbnailDataUrl.value = null;

    try {
      const isImage = isImageFile(file);
      const category: 'chat-image' | 'file' = isImage ? 'chat-image' : 'file';

      // 图片：生成本地预览（上传前显示）
      if (isImage) {
        thumbnailDataUrl.value = await generateLocalPreview(file);
      }

      // 上传原图
      uploadProgress.value = 30;
      const originalResult = await uploadFile(file, category);
      uploadProgress.value = 70;

      // 图片：上传缩略图到存储服务
      let thumbnailUrl: string | undefined;
      if (isImage) {
        const thumbnailBlob = await generateThumbnail(file);
        const thumbnailFile = new File([thumbnailBlob], `thumb_${file.name}`, {
          type: 'image/jpeg',
        });
        const thumbResult = await uploadFile(thumbnailFile, 'chat-image');
        thumbnailUrl = thumbResult.publicUrl;
      }

      uploadProgress.value = 100;

      const result: FileMessageContent = {
        file_id: originalResult.fileId,
        file_name: file.name,
        file_size: file.size,
        content_type: file.type,
        public_url: originalResult.publicUrl,
        category,
        thumbnail_url: thumbnailUrl,
      };

      fileData.value = result;
      return result;
    } catch (err: any) {
      error.value = err.message || '上传失败，请重试';
      message.error(error.value);
      return null;
    } finally {
      uploading.value = false;
    }
  }

  function clearFile() {
    fileData.value = null;
    uploading.value = false;
    uploadProgress.value = 0;
    error.value = null;
    thumbnailDataUrl.value = null;
  }

  return {
    uploading,
    uploadProgress,
    error,
    fileData,
    thumbnailDataUrl,
    isImageFile,
    validateFile,
    processAndUpload,
    clearFile,
  };
}
