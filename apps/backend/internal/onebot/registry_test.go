package onebot

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"purr-chat-server/internal/models"
)

func TestActionAliasesResolveToCanonicalDefinition(t *testing.T) {
	t.Parallel()

	canonical, err := ResolveAction("send_message")
	require.NoError(t, err)
	for _, alias := range []string{"send_msg", "send_private_msg", "send_group_msg"} {
		definition, aliasErr := ResolveAction(alias)
		require.NoError(t, aliasErr)
		assert.Equal(t, canonical, definition)
	}
}

func TestRegistryCapabilitiesAreKnownAndCopiesAreImmutable(t *testing.T) {
	t.Parallel()

	actions := Actions()
	require.NotEmpty(t, actions)
	for _, action := range actions {
		if action.RequiredCapability != "" {
			assert.Contains(t, models.AllCapabilities, action.RequiredCapability, action.Name)
		}
	}

	actions[0].Name = "mutated"
	actions[0].Aliases[0] = "mutated"
	definition, ok := LookupAction("send_message")
	require.True(t, ok)
	assert.Equal(t, "send_message", definition.Name)
	assert.NotContains(t, definition.Aliases, "mutated")
}

func TestProfileAndSegmentSchemasAreMachineReadable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "opaque_string", Profile().IDFormat)
	assert.Equal(t, "conversation_id", Profile().ConversationKey)
	assert.Equal(t, "segment_array", Profile().MessageFormat)
	assert.False(t, Profile().CQCodeCoreFormat)

	segments := Segments()
	require.Len(t, segments, 5)
	for _, segment := range segments {
		assert.NotEmpty(t, segment.Fields, segment.Type)
	}

	segments[0].Fields[0].Name = "mutated"
	text, ok := LookupSegment("text")
	require.True(t, ok)
	assert.Equal(t, "text", text.Fields[0].Name)
}

func TestRegistryRejectsDuplicateNamesAndAliases(t *testing.T) {
	t.Parallel()

	definitions := []ActionDefinition{
		{Name: "first", Aliases: []string{"shared"}, Version: ProfileVersion, Transports: allTransports()},
		{Name: "shared", Version: ProfileVersion, Transports: allTransports()},
	}
	assert.Panics(t, func() { mustBuildActionIndex(definitions) })
}

func TestUnknownAndRejectedActionsReturnStandardErrors(t *testing.T) {
	t.Parallel()

	_, err := ResolveAction("does_not_exist")
	require.Error(t, err)
	assert.Equal(t, RetCodeUnknownAction, AsError(err).Code)

	for _, action := range []string{"get_cookies", "get_csrf_token", "get_credentials", "get_rkey", "upload_file_by_path", "upload_group_file", "upload_private_file"} {
		_, rejectedErr := ResolveAction(action)
		require.Error(t, rejectedErr)
		assert.Equal(t, RetCodePermissionDenied, AsError(rejectedErr).Code)
	}
}

func TestRetCodePartitions(t *testing.T) {
	t.Parallel()

	tests := map[RetCode]string{
		RetCodeOK:               "success",
		RetCodeBadRequest:       "request",
		RetCodeUnauthenticated:  "authentication",
		RetCodePermissionDenied: "permission",
		RetCodeResourceNotFound: "resource",
		RetCodeRateLimited:      "rate_limit",
		RetCodeInternal:         "internal",
	}
	for code, category := range tests {
		assert.Equal(t, category, code.Category())
	}
}

func TestMessageSegmentSchemasAndSupport(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`[
		{"type":"text","data":{"text":"hello"}},
		{"type":"image","data":{"file_id":"file_opaque"}},
		{"type":"file","data":{"url":"https://example.invalid/file","name":"notes.txt"}},
		{"type":"at","data":{"user_id":"user_opaque"}},
		{"type":"reply","data":{"message_id":"message_opaque"}}
	]`)
	segments, err := DecodeMessageSegments(payload)
	require.NoError(t, err)
	require.Len(t, segments, 5)

	err = RequireStableSegments(segments)
	require.Error(t, err)
	assert.Equal(t, RetCodeUnsupportedSegment, AsError(err).Code)

	stable, err := DecodeMessageSegments(json.RawMessage(`[{"type":"text","data":{"text":"hello"}}]`))
	require.NoError(t, err)
	assert.NoError(t, RequireStableSegments(stable))
}

func TestMessageSegmentsRejectUnknownInvalidAndLocalFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		code    RetCode
	}{
		{name: "unknown", payload: `[{"type":"video","data":{}}]`, code: RetCodeUnsupportedSegment},
		{name: "empty text", payload: `[{"type":"text","data":{"text":""}}]`, code: RetCodeInvalidParams},
		{name: "unknown text data", payload: `[{"type":"text","data":{"text":"hi","secret":true}}]`, code: RetCodeInvalidParams},
		{name: "trailing value", payload: `[{"type":"text","data":{"text":"hi"}}] []`, code: RetCodeInvalidParams},
		{name: "local absolute path", payload: `[{"type":"file","data":{"file_id":"/etc/passwd"}}]`, code: RetCodeInvalidParams},
		{name: "local file ID", payload: `[{"type":"image","data":{"file_id":"file:///etc/passwd"}}]`, code: RetCodeInvalidParams},
		{name: "local file URL", payload: `[{"type":"image","data":{"url":"file:///etc/passwd"}}]`, code: RetCodeInvalidParams},
		{name: "insecure URL", payload: `[{"type":"image","data":{"url":"http://example.invalid/file"}}]`, code: RetCodeInvalidParams},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeMessageSegments(json.RawMessage(test.payload))
			require.Error(t, err)
			assert.Equal(t, test.code, AsError(err).Code)
		})
	}
}
