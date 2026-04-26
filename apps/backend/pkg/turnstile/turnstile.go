package turnstile

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

const verifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// VerifyResponse Turnstile 验证响应
type VerifyResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

// Verify 验证 Turnstile token
func Verify(secretKey, token, remoteIP string) (*VerifyResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	data := url.Values{
		"secret":   {secretKey},
		"response": {token},
	}
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	resp, err := client.PostForm(verifyURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result VerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
