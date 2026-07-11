package onebot

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"purr-chat-server/internal/models"
)

type ImplementationStatus string

const (
	StatusStable        ImplementationStatus = "stable"
	StatusPartial       ImplementationStatus = "partial"
	StatusPlanned       ImplementationStatus = "planned"
	StatusBlocked       ImplementationStatus = "blocked"
	StatusNotApplicable ImplementationStatus = "not_applicable"
	StatusRejected      ImplementationStatus = "rejected"
)

type Transport string

const (
	TransportUniversalWebSocket Transport = "universal_websocket"
	TransportHTTP               Transport = "http"
)

type ProfileDefinition struct {
	Version          string `json:"version"`
	IDFormat         string `json:"id_format"`
	ConversationKey  string `json:"conversation_key"`
	MessageFormat    string `json:"message_format"`
	CQCodeCoreFormat bool   `json:"cq_code_core_format"`
}

type ActionDefinition struct {
	Name               string               `json:"name"`
	Aliases            []string             `json:"aliases,omitempty"`
	Category           string               `json:"category"`
	Status             ImplementationStatus `json:"status"`
	Transports         []Transport          `json:"transports"`
	RequiredCapability string               `json:"required_capability,omitempty"`
	Version            string               `json:"version"`
	CompatibilityNote  string               `json:"compatibility_note,omitempty"`
}

type SegmentDefinition struct {
	Type              string                      `json:"type"`
	Status            ImplementationStatus        `json:"status"`
	CompatibilityNote string                      `json:"compatibility_note,omitempty"`
	Fields            []SegmentField              `json:"fields"`
	Validate          func(json.RawMessage) error `json:"-"`
}

type SegmentField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

var profile = ProfileDefinition{
	Version:          ProfileVersion,
	IDFormat:         "opaque_string",
	ConversationKey:  "conversation_id",
	MessageFormat:    "segment_array",
	CQCodeCoreFormat: false,
}

var actionDefinitions = []ActionDefinition{
	{Name: "send_message", Aliases: []string{"send_msg", "send_private_msg", "send_group_msg"}, Category: "message", Status: StatusPlanned, Transports: allTransports(), RequiredCapability: models.CapabilitySend, Version: ProfileVersion, CompatibilityNote: "conversation_id is the routing key; private/group aliases do not imply numeric QQ IDs"},
	{Name: "get_message_history", Aliases: []string{"get_group_msg_history"}, Category: "message", Status: StatusPlanned, Transports: allTransports(), RequiredCapability: models.CapabilityReadHistory, Version: ProfileVersion},
	{Name: "get_conversation_info", Aliases: []string{"get_group_info"}, Category: "conversation", Status: StatusPlanned, Transports: allTransports(), Version: ProfileVersion},
	{Name: "get_conversation_member_list", Aliases: []string{"get_group_member_list"}, Category: "member", Status: StatusPlanned, Transports: allTransports(), RequiredCapability: models.CapabilityMembersRead, Version: ProfileVersion},
	{Name: "get_login_info", Category: "bot", Status: StatusPlanned, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "returns the authenticated PurrChat Bot identity"},
	{Name: "get_cookies", Category: "credential", Status: StatusRejected, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "PurrChat never exposes account cookies"},
	{Name: "get_csrf_token", Category: "credential", Status: StatusRejected, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "PurrChat never exposes CSRF credentials"},
	{Name: "get_credentials", Category: "credential", Status: StatusRejected, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "PurrChat never exposes credentials"},
	{Name: "get_rkey", Category: "credential", Status: StatusRejected, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "QQ/NapCat rkey is not applicable to PurrChat"},
	{Name: "upload_file_by_path", Category: "file", Status: StatusRejected, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "arbitrary server-local file reads are forbidden"},
	{Name: "upload_group_file", Category: "file", Status: StatusRejected, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "NapCat local path upload is forbidden; managed file IDs may be supported by a future PurrChat action"},
	{Name: "upload_private_file", Category: "file", Status: StatusRejected, Transports: allTransports(), Version: ProfileVersion, CompatibilityNote: "NapCat local path upload is forbidden; managed file IDs may be supported by a future PurrChat action"},
}

var segmentDefinitions = []SegmentDefinition{
	{Type: "text", Status: StatusStable, Fields: []SegmentField{{Name: "text", Type: "string", Required: true}}, Validate: validateText},
	{Type: "image", Status: StatusPartial, CompatibilityNote: "schema reserved; sending is rejected until managed media ingestion is available", Fields: fileReferenceFields(), Validate: validateImage},
	{Type: "file", Status: StatusPartial, CompatibilityNote: "schema reserved; local file paths are never accepted", Fields: append(fileReferenceFields(), SegmentField{Name: "name", Type: "string"}), Validate: validateFile},
	{Type: "at", Status: StatusPartial, CompatibilityNote: "schema reserved; mention rendering is not implemented", Fields: []SegmentField{{Name: "user_id", Type: "opaque_string", Required: true}}, Validate: validateAt},
	{Type: "reply", Status: StatusPartial, CompatibilityNote: "schema reserved; reply persistence is not implemented", Fields: []SegmentField{{Name: "message_id", Type: "opaque_string", Required: true}}, Validate: validateReply},
}

var (
	actionsByName  = mustBuildActionIndex(actionDefinitions)
	segmentsByType = mustBuildSegmentIndex(segmentDefinitions)
)

func Actions() []ActionDefinition {
	result := make([]ActionDefinition, len(actionDefinitions))
	for i, definition := range actionDefinitions {
		result[i] = cloneActionDefinition(definition)
	}
	return result
}

func Profile() ProfileDefinition {
	return profile
}

func LookupAction(name string) (ActionDefinition, bool) {
	definition, ok := actionsByName[name]
	return cloneActionDefinition(definition), ok
}

func ResolveAction(name string) (ActionDefinition, error) {
	definition, ok := LookupAction(name)
	if !ok {
		return ActionDefinition{}, NewError(RetCodeUnknownAction, "unknown action: "+name, nil)
	}
	if definition.Status == StatusRejected {
		return ActionDefinition{}, NewError(RetCodePermissionDenied, "action is rejected by the PurrChat OneBot profile: "+name, nil)
	}
	return definition, nil
}

func Segments() []SegmentDefinition {
	result := make([]SegmentDefinition, len(segmentDefinitions))
	for i, definition := range segmentDefinitions {
		result[i] = cloneSegmentDefinition(definition)
	}
	return result
}

func LookupSegment(segmentType string) (SegmentDefinition, bool) {
	definition, ok := segmentsByType[segmentType]
	return cloneSegmentDefinition(definition), ok
}

func mustBuildActionIndex(definitions []ActionDefinition) map[string]ActionDefinition {
	index := make(map[string]ActionDefinition)
	for _, definition := range definitions {
		if definition.Name == "" || definition.Version == "" || len(definition.Transports) == 0 {
			panic("onebot: incomplete action definition")
		}
		for _, name := range append([]string{definition.Name}, definition.Aliases...) {
			if _, exists := index[name]; exists {
				panic("onebot: duplicate action or alias: " + name)
			}
			index[name] = definition
		}
	}
	return index
}

func mustBuildSegmentIndex(definitions []SegmentDefinition) map[string]SegmentDefinition {
	index := make(map[string]SegmentDefinition)
	for _, definition := range definitions {
		if definition.Type == "" || definition.Validate == nil {
			panic("onebot: incomplete segment definition")
		}
		if _, exists := index[definition.Type]; exists {
			panic("onebot: duplicate segment: " + definition.Type)
		}
		index[definition.Type] = definition
	}
	return index
}

func allTransports() []Transport {
	return []Transport{TransportUniversalWebSocket, TransportHTTP}
}

func cloneActionDefinition(definition ActionDefinition) ActionDefinition {
	definition.Aliases = slices.Clone(definition.Aliases)
	definition.Transports = slices.Clone(definition.Transports)
	return definition
}

func cloneSegmentDefinition(definition SegmentDefinition) SegmentDefinition {
	definition.Fields = slices.Clone(definition.Fields)
	return definition
}

func fileReferenceFields() []SegmentField {
	return []SegmentField{
		{Name: "file_id", Type: "opaque_string"},
		{Name: "url", Type: "https_url"},
	}
}

func validateText(data json.RawMessage) error {
	value, err := decodeSegmentData[TextData]("text", data)
	if err != nil {
		return err
	}
	if value.Text == "" {
		return NewError(RetCodeInvalidParams, "text segment requires non-empty text", nil)
	}
	return nil
}

func validateImage(data json.RawMessage) error {
	value, err := decodeSegmentData[ImageData]("image", data)
	if err != nil {
		return err
	}
	return validateManagedFileReference("image", value.FileID, value.URL)
}

func validateFile(data json.RawMessage) error {
	value, err := decodeSegmentData[FileData]("file", data)
	if err != nil {
		return err
	}
	return validateManagedFileReference("file", value.FileID, value.URL)
}

func validateAt(data json.RawMessage) error {
	value, err := decodeSegmentData[AtData]("at", data)
	if err != nil {
		return err
	}
	return ValidateOpaqueID("user_id", value.UserID)
}

func validateReply(data json.RawMessage) error {
	value, err := decodeSegmentData[ReplyData]("reply", data)
	if err != nil {
		return err
	}
	return ValidateOpaqueID("message_id", value.MessageID)
}

func validateManagedFileReference(segmentType, fileID, rawURL string) error {
	if fileID == "" && rawURL == "" {
		return NewError(RetCodeInvalidParams, fmt.Sprintf("%s segment requires file_id or url", segmentType), nil)
	}
	if strings.HasPrefix(fileID, "/") || strings.HasPrefix(fileID, "file:") {
		return NewError(RetCodeInvalidParams, "local file paths are forbidden", nil)
	}
	if rawURL != "" {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return NewError(RetCodeInvalidParams, "media url must be an absolute HTTPS URL", err)
		}
	}
	return nil
}
