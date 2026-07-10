package handlers

import (
	"net/http"
	"strconv"
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
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	resp, err := h.workflowService.GetWorkflow(c.Request.Context(), botID, userID)
	if err != nil {
		logger.ErrorfWithCaller("[WorkflowHandler] GetWorkflow failed: %v", err)
		if isAuthError(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}

	c.Header("ETag", resp.ETag)
	c.JSON(http.StatusOK, resp)
}

func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	botID := c.Param("id")
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req models.UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ifMatch := c.GetHeader("If-Match")
	if ifMatch != "" {
		etagRevision, err := parseETagRevision(ifMatch)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid If-Match revision"})
			return
		}
		if etagRevision != req.Revision {
			c.JSON(http.StatusConflict, gin.H{"error": "revision mismatch between If-Match and request body", "code": "revision_mismatch"})
			return
		}
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
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req models.PublishWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ifMatch := c.GetHeader("If-Match"); ifMatch != "" {
		etagRevision, err := parseETagRevision(ifMatch)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid If-Match revision"})
			return
		}
		if etagRevision != req.Revision {
			c.JSON(http.StatusConflict, gin.H{"error": "revision mismatch between If-Match and request body", "code": "revision_mismatch"})
			return
		}
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
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var req models.TestRunWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.workflowService.TestRunWorkflow(c.Request.Context(), botID, userID, &req)
	if err != nil {
		if isAuthError(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *WorkflowHandler) TestRunStep(c *gin.Context) {
	botID := c.Param("id")
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.workflowService.TestRunStep(c.Request.Context(), botID, userID, req.SessionID)
	if err != nil {
		if isAuthError(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *WorkflowHandler) ListPublishedVersions(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	versions, err := h.workflowService.ListPublishedVersions(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		if isAuthError(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

func (h *WorkflowHandler) RollbackWorkflow(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	revision, err := strconv.Atoi(c.Param("revision"))
	if err != nil || revision < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workflow revision"})
		return
	}
	resp, err := h.workflowService.RollbackWorkflow(c.Request.Context(), c.Param("id"), userID, revision)
	if err != nil {
		if isAuthError(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if isRevisionMismatch(err) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "code": "revision_mismatch"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow version not found"})
		return
	}
	c.Header("ETag", resp.ETag)
	c.JSON(http.StatusOK, resp)
}

func parseETagRevision(value string) (int, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "W/")
	value = strings.Trim(value, `"`)
	return strconv.Atoi(value)
}

func isRevisionMismatch(err error) bool {
	return strings.Contains(err.Error(), "revision mismatch")
}

func isAuthError(err error) bool {
	return strings.Contains(err.Error(), "not authorized")
}
