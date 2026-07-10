package handlers

import (
	"net/http"
	"strings"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

type WorkflowHandler struct {
	workflowService *services.WorkflowService
}

func NewWorkflowHandler(ws *services.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{workflowService: ws}
}

func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	botID := c.Param("id")

	resp, err := h.workflowService.GetWorkflow(c.Request.Context(), botID)
	if err != nil {
		logger.ErrorfWithCaller("[WorkflowHandler] GetWorkflow failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}

	c.Header("ETag", resp.ETag)
	c.JSON(http.StatusOK, resp)
}

func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString("userID")

	var req models.UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ifMatch := c.GetHeader("If-Match")
	if ifMatch != "" {
		etagRev := strings.Trim(ifMatch, `"`)
		_ = etagRev
	}

	resp, err := h.workflowService.UpdateWorkflow(c.Request.Context(), botID, userID, &req)
	if err != nil {
		if isRevisionMismatch(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "revision_mismatch"})
			return
		}
		if isAuthError(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("ETag", resp.ETag)
	c.JSON(http.StatusOK, resp)
}

func (h *WorkflowHandler) ValidateWorkflow(c *gin.Context) {
	var req models.ValidateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.workflowService.ValidateWorkflow(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *WorkflowHandler) PublishWorkflow(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString("userID")

	var req models.PublishWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version, err := h.workflowService.PublishWorkflow(c.Request.Context(), botID, userID, &req)
	if err != nil {
		if isAuthError(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if isRevisionMismatch(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "revision_mismatch"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, version)
}

func (h *WorkflowHandler) TestRunWorkflow(c *gin.Context) {
	botID := c.Param("id")

	var req models.TestRunWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.workflowService.TestRunWorkflow(c.Request.Context(), botID, &req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *WorkflowHandler) TestRunStep(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.workflowService.TestRunStep(c.Request.Context(), req.SessionID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func isRevisionMismatch(err error) bool {
	return strings.Contains(err.Error(), "revision mismatch")
}

func isAuthError(err error) bool {
	return strings.Contains(err.Error(), "not authorized")
}
