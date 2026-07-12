package onebot

import (
	"bytes"
	"encoding/json"
	"io"
)

type MessageSegment struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type TextData struct {
	Text string `json:"text"`
}

type ImageData struct {
	FileID string `json:"file_id,omitempty"`
	URL    string `json:"url,omitempty"`
}

type FileData struct {
	FileID string `json:"file_id,omitempty"`
	URL    string `json:"url,omitempty"`
	Name   string `json:"name,omitempty"`
}

type AtData struct {
	UserID string `json:"user_id"`
}

type ReplyData struct {
	MessageID string `json:"message_id"`
}

func DecodeMessageSegments(payload json.RawMessage) ([]MessageSegment, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var segments []MessageSegment
	if err := decoder.Decode(&segments); err != nil {
		return nil, NewError(RetCodeInvalidParams, "message must be a segment array", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, NewError(RetCodeInvalidParams, "message must contain one segment array", err)
	}
	if segments == nil {
		return nil, NewError(RetCodeInvalidParams, "message must be a segment array", nil)
	}
	for i := range segments {
		definition, ok := LookupSegment(segments[i].Type)
		if !ok {
			return nil, NewError(RetCodeUnsupportedSegment, "unsupported message segment: "+segments[i].Type, nil)
		}
		if err := definition.Validate(segments[i].Data); err != nil {
			return nil, err
		}
	}
	return segments, nil
}

func RequireStableSegments(segments []MessageSegment) error {
	for _, segment := range segments {
		definition, ok := LookupSegment(segment.Type)
		if !ok || definition.Status != StatusStable {
			return NewError(RetCodeUnsupportedSegment, "message segment is not supported by this action: "+segment.Type, nil)
		}
	}
	return nil
}

func decodeSegmentData[T any](segmentType string, data json.RawMessage) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, NewError(RetCodeInvalidParams, "invalid "+segmentType+" segment data", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return value, NewError(RetCodeInvalidParams, segmentType+" segment data must contain one JSON value", err)
	}
	return value, nil
}
