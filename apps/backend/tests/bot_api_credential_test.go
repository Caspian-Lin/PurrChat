package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"purr-chat-server/internal/handlers"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type credentialDisconnectSpy struct{ ids []uuid.UUID }

func (s *credentialDisconnectSpy) DisconnectCredential(_ context.Context, id uuid.UUID) error {
	s.ids = append(s.ids, id)
	return nil
}

func setupCredentialService(t *testing.T) (*services.BotAPICredentialService, repository.BotAPICredentialRepository, repository.BotRepository, *credentialDisconnectSpy) {
	t.Helper()
	SetupTestDB(t)
	t.Cleanup(func() { CleanupTestDB(t) })
	repo := repository.NewBotAPICredentialRepository()
	botRepo := repository.NewBotRepository()
	spy := &credentialDisconnectSpy{}
	return services.NewBotAPICredentialService(repo, botRepo, spy), repo, botRepo, spy
}

func createExternalBot(t *testing.T, botRepo repository.BotRepository, ownerID uuid.UUID, name string) *models.Bot {
	t.Helper()
	bot := &models.Bot{OwnerID: ownerID, Name: name, Status: models.BotStatusActive, BotType: models.BotTypeExternal}
	require.NoError(t, botRepo.Create(context.Background(), bot))
	return bot
}

func TestBotAPICredentialLifecycleAndTokenSecrecy(t *testing.T) {
	service, repo, botRepo, disconnect := setupCredentialService(t)
	ctx := context.Background()
	owner := CreateTestUser(t, "cred_owner", "cred_owner@test.com", "pass")
	bot := createExternalBot(t, botRepo, owner.ID, "External API Bot")

	first, err := service.Create(ctx, owner.ID, bot.ID, "development", nil)
	require.NoError(t, err)
	second, err := service.Create(ctx, owner.ID, bot.ID, "production", nil)
	require.NoError(t, err)
	require.NotEqual(t, first.Token, second.Token)
	assert.True(t, strings.HasPrefix(first.Token, "purr_bot_"))
	assert.GreaterOrEqual(t, len(first.Token), 50)

	items, err := service.List(ctx, owner.ID, bot.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	listed, err := json.Marshal(items)
	require.NoError(t, err)
	assert.NotContains(t, string(listed), first.Token)
	assert.NotContains(t, string(listed), second.Token)

	var storedHash []byte
	require.NoError(t, database.GetPool().QueryRow(ctx, `SELECT token_hash FROM bot_api_credentials WHERE id=$1`, first.Credential.ID).Scan(&storedHash))
	assert.Len(t, storedHash, 32)
	assert.NotContains(t, string(storedHash), first.Token)

	principal, err := service.Authenticate(ctx, "Bearer "+first.Token)
	require.NoError(t, err)
	assert.Equal(t, bot.ID, principal.BotID)
	assert.Equal(t, bot.ID, principal.IdentityID)
	assert.Equal(t, first.Credential.ID, principal.CredentialID)
	used, err := repo.FindByID(ctx, first.Credential.ID)
	require.NoError(t, err)
	assert.NotNil(t, used.LastUsedAt)

	rotated, err := service.Rotate(ctx, owner.ID, bot.ID, first.Credential.ID)
	require.NoError(t, err)
	assert.NotEqual(t, first.Token, rotated.Token)
	_, err = service.Authenticate(ctx, "Bearer "+first.Token)
	assert.ErrorIs(t, err, services.ErrCredentialInvalid)
	_, err = service.Authenticate(ctx, "Bearer "+rotated.Token)
	require.NoError(t, err)

	_, err = service.Revoke(ctx, owner.ID, bot.ID, first.Credential.ID)
	require.NoError(t, err)
	_, err = service.Authenticate(ctx, "Bearer "+rotated.Token)
	assert.ErrorIs(t, err, services.ErrCredentialRevoked)
	assert.Equal(t, []uuid.UUID{first.Credential.ID, first.Credential.ID}, disconnect.ids)

	var auditText string
	require.NoError(t, database.GetPool().QueryRow(ctx, `SELECT COALESCE(string_agg(metadata::text, ''), '') FROM bot_api_credential_audit_logs WHERE bot_id=$1`, bot.ID).Scan(&auditText))
	assert.NotContains(t, auditText, first.Token)
	assert.NotContains(t, auditText, rotated.Token)
	var events int
	require.NoError(t, database.GetPool().QueryRow(ctx, `SELECT COUNT(*) FROM bot_api_credential_audit_logs WHERE credential_id=$1`, first.Credential.ID).Scan(&events))
	assert.Equal(t, 3, events)
}

func TestBotAPICredentialAuthorizationAndStateValidation(t *testing.T) {
	service, _, botRepo, _ := setupCredentialService(t)
	ctx := context.Background()
	owner := CreateTestUser(t, "cred_auth_owner", "cred_auth_owner@test.com", "pass")
	other := CreateTestUser(t, "cred_auth_other", "cred_auth_other@test.com", "pass")
	external := createExternalBot(t, botRepo, owner.ID, "External State Bot")

	_, err := service.Create(ctx, other.ID, external.ID, "forbidden", nil)
	assert.ErrorIs(t, err, services.ErrCredentialForbidden)
	workflow := &models.Bot{OwnerID: owner.ID, Name: "Workflow State Bot", Status: models.BotStatusActive, BotType: models.BotTypeWorkflow}
	require.NoError(t, botRepo.Create(ctx, workflow))
	_, err = service.Create(ctx, owner.ID, workflow.ID, "wrong type", nil)
	assert.ErrorIs(t, err, services.ErrBotNotExternal)

	expiredAt := time.Now().Add(time.Hour)
	expired, err := service.Create(ctx, owner.ID, external.ID, "expires", &expiredAt)
	require.NoError(t, err)
	tag, err := database.GetPool().Exec(ctx, `UPDATE bot_api_credentials SET expires_at=$2 WHERE id=$1`, expired.Credential.ID, time.Now().UTC().Add(-time.Minute))
	require.NoError(t, err)
	require.EqualValues(t, 1, tag.RowsAffected())
	storedExpired, err := repository.NewBotAPICredentialRepository().FindByID(ctx, expired.Credential.ID)
	require.NoError(t, err)
	require.True(t, storedExpired.ExpiresAt.Before(time.Now()), "stored expiry: %s", storedExpired.ExpiresAt)
	_, err = service.Authenticate(ctx, "Bearer "+expired.Token)
	assert.ErrorIs(t, err, services.ErrCredentialExpired)

	active, err := service.Create(ctx, owner.ID, external.ID, "disabled bot", nil)
	require.NoError(t, err)
	external.Status = models.BotStatusDisabled
	require.NoError(t, botRepo.Update(ctx, external))
	_, err = service.Authenticate(ctx, "Bearer "+active.Token)
	assert.ErrorIs(t, err, services.ErrBotDisabled)
	external.Status = models.BotStatusActive
	require.NoError(t, botRepo.Update(ctx, external))
	_, err = database.GetPool().Exec(ctx, `DELETE FROM bot_identities WHERE app_id=$1`, external.ID)
	require.NoError(t, err)
	_, err = service.Authenticate(ctx, "Bearer "+active.Token)
	assert.ErrorIs(t, err, services.ErrBotIdentityMissing)
}

func TestBotCredentialAuthMiddlewareIsStrictBearerOnly(t *testing.T) {
	service, _, botRepo, _ := setupCredentialService(t)
	owner := CreateTestUser(t, "cred_bearer_owner", "cred_bearer_owner@test.com", "pass")
	bot := createExternalBot(t, botRepo, owner.ID, "Bearer Bot")
	secret, err := service.Create(context.Background(), owner.ID, bot.ID, "strict", nil)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/bot", handlers.BotCredentialAuthMiddleware(service), func(c *gin.Context) {
		principal, exists := c.Get(handlers.BotPrincipalContextKey)
		require.True(t, exists)
		c.JSON(http.StatusOK, principal)
	})

	for name, mutate := range map[string]func(*http.Request){
		"query":  func(r *http.Request) { r.URL.RawQuery = "token=" + secret.Token },
		"cookie": func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "token", Value: secret.Token}) },
		"basic":  func(r *http.Request) { r.Header.Set("Authorization", "Basic "+secret.Token) },
		"lower":  func(r *http.Request) { r.Header.Set("Authorization", "bearer "+secret.Token) },
		"spaces": func(r *http.Request) { r.Header.Set("Authorization", "Bearer  "+secret.Token) },
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/bot", nil)
			mutate(req)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			assert.Equal(t, http.StatusUnauthorized, res.Code)
			assert.NotContains(t, res.Body.String(), secret.Token)
		})
	}
	req := httptest.NewRequest(http.MethodGet, "/bot?token=wrong", nil)
	req.Header.Set("Authorization", "Bearer "+secret.Token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotContains(t, res.Body.String(), secret.Token)
}

func TestBotCredentialOwnerManagementAPI(t *testing.T) {
	service, _, botRepo, _ := setupCredentialService(t)
	owner := CreateTestUser(t, "cred_api_owner", "cred_api_owner@test.com", "pass")
	other := CreateTestUser(t, "cred_api_other", "cred_api_other@test.com", "pass")
	bot := createExternalBot(t, botRepo, owner.ID, "Management Bot")
	handler := handlers.NewBotAPICredentialHandler(service)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/bots", handlers.AuthMiddleware(jwtSecret))
	group.POST("/:id/credentials", handler.Create)
	group.GET("/:id/credentials", handler.List)
	group.POST("/:id/credentials/:credential_id/rotate", handler.Rotate)
	group.DELETE("/:id/credentials/:credential_id", handler.Revoke)

	body := bytes.NewBufferString(`{"name":"api production"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/bots/"+bot.ID.String()+"/credentials", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+GetAuthToken(t, owner.ID.String()))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, http.StatusCreated, res.Code)
	var created models.BotAPICredentialSecret
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &created))
	assert.NotEmpty(t, created.Token)

	req = httptest.NewRequest(http.MethodGet, "/api/bots/"+bot.ID.String()+"/credentials", nil)
	req.Header.Set("Authorization", "Bearer "+GetAuthToken(t, owner.ID.String()))
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotContains(t, res.Body.String(), created.Token)

	req = httptest.NewRequest(http.MethodGet, "/api/bots/"+bot.ID.String()+"/credentials", nil)
	req.Header.Set("Authorization", "Bearer "+GetAuthToken(t, other.ID.String()))
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusForbidden, res.Code)
	assert.NotContains(t, res.Body.String(), created.Token)
}
