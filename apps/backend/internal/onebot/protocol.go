package onebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const ProfileVersion = "purrchat-onebot-v1"

type ActionRequest struct {
	Action string          `json:"action"`
	Params json.RawMessage `json:"params"`
	Echo   json.RawMessage `json:"echo,omitempty"`
}

type ActionResponse struct {
	Status  ResponseStatus  `json:"status"`
	RetCode RetCode         `json:"retcode"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Echo    json.RawMessage `json:"echo,omitempty"`
	TraceID string          `json:"trace_id"`
}

type Event struct {
	Time       int64           `json:"time"`
	SelfID     string          `json:"self_id"`
	PostType   string          `json:"post_type"`
	EventID    string          `json:"event_id"`
	DetailType string          `json:"detail_type"`
	SubType    string          `json:"sub_type,omitempty"`
	Data       json.RawMessage `json:"data"`
}

type ResponseStatus string

const (
	StatusOK     ResponseStatus = "ok"
	StatusFailed ResponseStatus = "failed"
)

func DecodeActionRequest(payload []byte) (ActionRequest, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var request ActionRequest
	if err := decoder.Decode(&request); err != nil {
		return ActionRequest{}, NewError(RetCodeBadRequest, "invalid action request", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ActionRequest{}, NewError(RetCodeBadRequest, "action request must contain one JSON value", nil)
	}
	if request.Action == "" {
		return ActionRequest{}, NewError(RetCodeInvalidParams, "action is required", nil)
	}
	if len(request.Params) == 0 {
		request.Params = json.RawMessage(`{}`)
	}
	if first := firstJSONToken(request.Params); first != '{' {
		return ActionRequest{}, NewError(RetCodeInvalidParams, "params must be an object", nil)
	}

	return request, nil
}

func Success(data, echo json.RawMessage, traceID string) ActionResponse {
	if len(data) == 0 {
		data = json.RawMessage(`null`)
	}
	return ActionResponse{
		Status:  StatusOK,
		RetCode: RetCodeOK,
		Data:    cloneRawMessage(data),
		Echo:    cloneRawMessage(echo),
		TraceID: traceID,
	}
}

func Failure(err error, echo json.RawMessage, traceID string) ActionResponse {
	protocolErr := AsError(err)
	return ActionResponse{
		Status:  StatusFailed,
		RetCode: protocolErr.Code,
		Data:    json.RawMessage(`null`),
		Message: protocolErr.Message,
		Echo:    cloneRawMessage(echo),
		TraceID: traceID,
	}
}

func DecodeParams[T any](request ActionRequest) (T, error) {
	var params T
	decoder := json.NewDecoder(bytes.NewReader(request.Params))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&params); err != nil {
		return params, NewError(RetCodeInvalidParams, "invalid action params", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return params, NewError(RetCodeInvalidParams, "action params must contain one JSON value", err)
	}
	return params, nil
}

func firstJSONToken(value json.RawMessage) byte {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return 0
	}
	return trimmed[0]
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func ValidateOpaqueID(name, value string) error {
	if value == "" {
		return NewError(RetCodeInvalidParams, fmt.Sprintf("%s is required", name), nil)
	}
	return nil
}
