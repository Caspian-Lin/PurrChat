package websocket

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewHub 测试创建Hub
func TestNewHub(t *testing.T) {
	hub := NewHub(100, 3)
	assert.NotNil(t, hub)
	assert.Equal(t, 100, hub.maxConnections)
	assert.Equal(t, 3, hub.maxUserConnections)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.userClients)
	assert.NotNil(t, hub.userDeviceClients)
}

// TestDeviceType 测试设备类型常量
func TestDeviceType(t *testing.T) {
	assert.Equal(t, DeviceType("unknown"), DeviceTypeUnknown)
	assert.Equal(t, DeviceType("web"), DeviceTypeWeb)
	assert.Equal(t, DeviceType("mobile"), DeviceTypeMobile)
	assert.Equal(t, DeviceType("desktop"), DeviceTypeDesktop)
	assert.Equal(t, DeviceType("tablet"), DeviceTypeTablet)
}

// TestClient 测试Client结构体
func TestClient(t *testing.T) {
	userID := uuid.New()
	client := &Client{
		ID:          uuid.New(),
		UserID:      userID,
		Conn:        nil,
		Send:        make(chan []byte, 256),
		DeviceType:  DeviceTypeWeb,
		ConnectedAt: time.Now(),
		UserAgent:   "test-agent",
	}

	assert.NotEqual(t, uuid.Nil, client.ID)
	assert.Equal(t, userID, client.UserID)
	assert.Equal(t, DeviceTypeWeb, client.DeviceType)
	assert.Equal(t, "test-agent", client.UserAgent)
	assert.NotNil(t, client.Send)
}

// TestBroadcastMessage 测试BroadcastMessage结构体
func TestBroadcastMessage(t *testing.T) {
	msg := BroadcastMessage{
		Type:      "test_type",
		Data:      "test_data",
		Timestamp: time.Now().Unix(),
	}

	assert.Equal(t, "test_type", msg.Type)
	assert.Equal(t, "test_data", msg.Data)
	assert.NotZero(t, msg.Timestamp)
}

// TestPrivateMessage 测试PrivateMessage结构体
func TestPrivateMessage(t *testing.T) {
	userID := uuid.New()
	msg := PrivateMessage{
		RecipientID: userID,
	}

	assert.Equal(t, userID, msg.RecipientID)
}
