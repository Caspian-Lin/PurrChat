/**
 * 通用文件上传工具 — 封装 requestUpload → PUT presigned URL → confirmUpload 三阶段流程
 * 被 useFileUpload 和 useAvatarUpload 共享
 */

import { storageApi } from '../models/api';

export interface UploadResult {
  fileId: string;
  publicUrl: string;
}

/**
 * 执行三阶段文件上传
 * 1. requestUpload — 获取预签名 URL
 * 2. PUT presigned URL — 上传文件到对象存储
 * 3. confirmUpload — 确认上传
 */
export async function threeStageUpload(
  file: File,
  category: string,
  usage: string
): Promise<UploadResult> {
  // 阶段 1: 申请上传
  const requestResp = await storageApi.requestUpload({
    file_name: file.name,
    file_size: file.size,
    content_type: file.type,
    category,
    usage,
  });
  if (!requestResp.success || !requestResp.data) {
    throw new Error(requestResp.message || '申请上传失败');
  }

  const { upload_id, object_key, upload_url } = requestResp.data;

  // 阶段 2: 使用预签名 URL 直接上传文件到对象存储
  const uploadResp = await fetch(upload_url, {
    method: 'PUT',
    body: file,
    headers: { 'Content-Type': file.type },
  });
  if (!uploadResp.ok) {
    throw new Error('文件上传失败');
  }

  // 阶段 3: 确认上传
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
