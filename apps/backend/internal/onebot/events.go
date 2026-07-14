package onebot

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	PostTypeMessage = "message"
	PostTypeNotice  = "notice"
	PostTypeMeta    = "meta_event"
)

const (
	NoticeGroupMemberIncrease    = "group_member_increase"
	NoticeGroupMemberDecrease    = "group_member_decrease"
	NoticeGroupMemberRoleChanged = "group_member_role_changed"
	NoticeInstallationChanged    = "purrchat_installation_changed"
)

const (
	SubTypeInstalled         = "installed"
	SubTypeSuspended         = "suspended"
	SubTypeResumed           = "resumed"
	SubTypeUninstalled       = "uninstalled"
	SubTypeCapabilityChanged = "capability_changed"
)

const DetailTypePrivate = "message.private"
const DetailTypeGroup = "message.group"

func GenerateEventID() string {
	return fmt.Sprintf("evt_%s", uuid.New().String())
}

func BuildMessageEvent(selfID, detailType string, timestamp time.Time, data any) (Event, error) {
	return buildEvent(selfID, PostTypeMessage, detailType, "", timestamp, data)
}

func BuildNoticeEvent(selfID, detailType, subType string, timestamp time.Time, data any) (Event, error) {
	return buildEvent(selfID, PostTypeNotice, detailType, subType, timestamp, data)
}

func buildEvent(selfID, postType, detailType, subType string, timestamp time.Time, data any) (Event, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Time:       timestamp.Unix(),
		SelfID:     selfID,
		PostType:   postType,
		EventID:    GenerateEventID(),
		DetailType: detailType,
		SubType:    subType,
		Data:       dataBytes,
	}, nil
}
