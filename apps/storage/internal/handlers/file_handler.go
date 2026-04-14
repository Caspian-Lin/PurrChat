package handlers

import (
	"net/http"

	"purr-chat-storage/internal/models"
	"purr-chat-storage/internal/services"
	"purr-chat-storage/pkg/logger"

	"github.com/gin-gonic/gin"
)

// FileHandler 文件上传下载处理器
type FileHandler struct {
	fileService *services.FileService
}

// NewFileHandler 创建文件处理器
func NewFileHandler(fileService *services.FileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

// RequestUpload 申请上传
// POST /api/files/upload/request
func (h *FileHandler) RequestUpload(c *gin.Context) {
	var req models.UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid upload request: %v", err)
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID := c.GetString("user_id")

	resp, err := h.fileService.RequestUpload(c.Request.Context(), userID, &req)
	if err != nil {
		logger.ErrorfWithCaller("Upload request failed for user %s: %v", userID, err)
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Upload requested: user=%s file=%s key=%s", userID, req.FileName, resp.ObjectKey)

	c.JSON(http.StatusOK, models.FileResponse{
		Success: true,
		Data:    resp,
	})
}

// ConfirmUpload 确认上传
// POST /api/files/upload/confirm
func (h *FileHandler) ConfirmUpload(c *gin.Context) {
	var req models.ConfirmUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid confirm request: %v", err)
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID := c.GetString("user_id")

	resp, err := h.fileService.ConfirmUpload(c.Request.Context(), userID, &req)
	if err != nil {
		logger.ErrorfWithCaller("Upload confirmation failed for user %s: %v", userID, err)
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Upload confirmed: user=%s file_id=%s key=%s", userID, resp.FileID, resp.ObjectKey)

	c.JSON(http.StatusOK, models.FileResponse{
		Success: true,
		Data:    resp,
	})
}

// GetDownloadURL 获取下载链接
// GET /api/files/download/url?file_id=xxx
func (h *FileHandler) GetDownloadURL(c *gin.Context) {
	fileID := c.Query("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: "file_id is required",
		})
		return
	}

	userID := c.GetString("user_id")

	resp, err := h.fileService.GetDownloadURL(c.Request.Context(), userID, fileID)
	if err != nil {
		logger.ErrorfWithCaller("Get download URL failed for user %s: %v", userID, err)
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.FileResponse{
		Success: true,
		Data:    resp,
	})
}

// DeleteFile 删除文件
// DELETE /api/files/:file_id
func (h *FileHandler) DeleteFile(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: "file_id is required",
		})
		return
	}

	userID := c.GetString("user_id")

	if err := h.fileService.DeleteFile(c.Request.Context(), userID, fileID); err != nil {
		logger.ErrorfWithCaller("Delete file failed for user %s: %v", userID, err)
		c.JSON(http.StatusBadRequest, models.FileResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("File deleted: user=%s file_id=%s", userID, fileID)

	c.JSON(http.StatusOK, models.FileResponse{
		Success: true,
		Message: "File deleted successfully",
	})
}
