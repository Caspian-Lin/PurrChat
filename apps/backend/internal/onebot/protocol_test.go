package onebot

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionRequestCodecPreservesEcho(t *testing.T) {
	t.Parallel()

	echoes := []string{
		`"request-1"`,
		`42`,
		`true`,
		`null`,
		`[1,"two",false]`,
		`{"nested":{"value":3}}`,
	}

	for _, echo := range echoes {
		t.Run(echo, func(t *testing.T) {
			t.Parallel()

			request, err := DecodeActionRequest([]byte(`{"action":"get_login_info","params":{},"echo":` + echo + `}`))
			require.NoError(t, err)

			response := Success(json.RawMessage(`{"user_id":"bot-1"}`), request.Echo, "trace-1")
			encoded, err := json.Marshal(response)
			require.NoError(t, err)

			var decoded map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			assert.JSONEq(t, echo, string(decoded["echo"]))
			assert.Equal(t, json.RawMessage(echo), request.Echo)
		})
	}
}

func TestDecodeActionRequestValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		code    RetCode
	}{
		{name: "malformed", payload: `{`, code: RetCodeBadRequest},
		{name: "unknown field", payload: `{"action":"x","params":{},"extra":true}`, code: RetCodeBadRequest},
		{name: "missing action", payload: `{"params":{}}`, code: RetCodeInvalidParams},
		{name: "non-object params", payload: `{"action":"x","params":[]}`, code: RetCodeInvalidParams},
		{name: "multiple values", payload: `{"action":"x","params":{}} {}`, code: RetCodeBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeActionRequest([]byte(test.payload))
			require.Error(t, err)
			assert.Equal(t, test.code, AsError(err).Code)
		})
	}
}

func TestDecodeParamsRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	type params struct {
		ConversationID string `json:"conversation_id"`
	}

	request := ActionRequest{Params: json.RawMessage(`{"conversation_id":"conversation-1","sender_id":"forged"}`)}
	_, err := DecodeParams[params](request)
	require.Error(t, err)
	assert.Equal(t, RetCodeInvalidParams, AsError(err).Code)

	request.Params = json.RawMessage(`{"conversation_id":"conversation-1"} {}`)
	_, err = DecodeParams[params](request)
	require.Error(t, err)
	assert.Equal(t, RetCodeInvalidParams, AsError(err).Code)
}

func TestEventCodecUsesOpaqueStringIDs(t *testing.T) {
	t.Parallel()

	event := Event{
		Time:       123,
		SelfID:     "bot_01J-opaque",
		PostType:   "message",
		EventID:    "evt_01J-opaque",
		DetailType: "group",
		Data:       json.RawMessage(`{"conversation_id":"conv_01J-opaque"}`),
	}
	encoded, err := json.Marshal(event)
	require.NoError(t, err)
	assert.JSONEq(t, `{"time":123,"self_id":"bot_01J-opaque","post_type":"message","event_id":"evt_01J-opaque","detail_type":"group","data":{"conversation_id":"conv_01J-opaque"}}`, string(encoded))
}

func TestFailureMapsProtocolAndInternalErrors(t *testing.T) {
	t.Parallel()

	protocolResponse := Failure(NewError(RetCodeRateLimited, "rate limit exceeded", nil), json.RawMessage(`7`), "trace-1")
	assert.Equal(t, RetCodeRateLimited, protocolResponse.RetCode)
	assert.Equal(t, "rate limit exceeded", protocolResponse.Message)

	internalResponse := Failure(errors.New("database password leaked here"), nil, "trace-2")
	assert.Equal(t, RetCodeInternal, internalResponse.RetCode)
	assert.Equal(t, "internal error", internalResponse.Message)
}
