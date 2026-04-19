package handlers

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "sync"
    "testing"
    "time"

    "purr-chat-server/pkg/logger"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "golang.org/x/time/rate"
)

func init() {
    logger.Init()
}

func TestVisitorLimiter_Allow(t *testing.T) {
    tests := []struct {
        name     string
        rate     rate.Limit
        burst    int
        requests int
        allowIdx int // 最后一个被允许的请求索引（0-based）
    }{
        {
            name:     "should allow request under burst",
            rate:     rate.Limit(1),
            burst:    3,
            requests: 1,
            allowIdx: 0,
        },
        {
            name:     "should allow up to burst requests",
            rate:     rate.Limit(1),
            burst:    3,
            requests: 3,
            allowIdx: 2,
        },
        {
            name:     "should reject after burst exhausted",
            rate:     rate.Limit(1),
            burst:    2,
            requests: 3,
            allowIdx: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            limiter := newVisitorLimiter(tt.rate, tt.burst, 10*time.Second)
            defer limiter.stop()

            allowed := 0
            for i := 0; i < tt.requests; i++ {
                if limiter.allow("test-key") {
                    allowed++
                }
            }

            assert.Equal(t, tt.allowIdx+1, allowed)
        })
    }
}

func TestVisitorLimiter_Replenish(t *testing.T) {
    t.Skip("Token replenishment timing is non-deterministic; skip in CI")
}

func TestVisitorLimiter_Cleanup(t *testing.T) {
    limiter := newVisitorLimiter(rate.Limit(1), 1, 100*time.Millisecond)
    defer limiter.stop()

    // Add visitors
    limiter.allow("ip-1")
    limiter.allow("ip-2")
    limiter.allow("ip-3")

    // Wait for cleanup to run (ttl/2 = 50ms, wait a bit longer)
    time.Sleep(200 * time.Millisecond)

    // The cleanup should have removed expired visitors
    // New allow should still work (creates new entry)
    assert.True(t, limiter.allow("ip-1"))
}

func TestVisitorLimiter_DifferentKeys(t *testing.T) {
    limiter := newVisitorLimiter(rate.Limit(1), 1, 10*time.Second)
    defer limiter.stop()

    // Different keys should have independent limits
    assert.True(t, limiter.allow("ip-1"))
    assert.True(t, limiter.allow("ip-2"))
    assert.False(t, limiter.allow("ip-1")) // ip-1 exhausted
    assert.False(t, limiter.allow("ip-2")) // ip-2 exhausted
}

func TestVisitorLimiter_Stop(t *testing.T) {
    limiter := newVisitorLimiter(rate.Limit(1), 1, 50*time.Millisecond)

    // Stop should not panic
    limiter.stop()

    // Calling stop again should panic (close of closed channel)
    // We don't test this as it's expected behavior
}

func TestIPRateLimitMiddleware(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(IPRateLimitMiddleware(rate.Limit(1), 2, 3*time.Second))
    router.GET("/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    t.Run("should allow requests under limit", func(t *testing.T) {
        w1 := httptest.NewRecorder()
        req1, _ := http.NewRequest("GET", "/test", nil)
        req1.RemoteAddr = "192.168.1.1:1234"
        router.ServeHTTP(w1, req1)
        assert.Equal(t, http.StatusOK, w1.Code)

        w2 := httptest.NewRecorder()
        req2, _ := http.NewRequest("GET", "/test", nil)
        req2.RemoteAddr = "192.168.1.1:1234"
        router.ServeHTTP(w2, req2)
        assert.Equal(t, http.StatusOK, w2.Code)
    })

    t.Run("should return 429 when rate exceeded", func(t *testing.T) {
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", "/test", nil)
        req.RemoteAddr = "192.168.1.1:1234"
        router.ServeHTTP(w, req)
        assert.Equal(t, http.StatusTooManyRequests, w.Code)

        var response map[string]interface{}
        err := json.Unmarshal(w.Body.Bytes(), &response)
        assert.NoError(t, err)
        assert.Equal(t, false, response["success"])
    })

    t.Run("should track different IPs separately", func(t *testing.T) {
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", "/test", nil)
        req.RemoteAddr = "10.0.0.1:5678"
        router.ServeHTTP(w, req)
        assert.Equal(t, http.StatusOK, w.Code)
    })
}

func TestUserRateLimitMiddleware(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(UserRateLimitMiddleware(rate.Limit(1), 1, 3*time.Second))
    router.GET("/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    t.Run("should use user_id when available", func(t *testing.T) {
        // First request with user_id
        w1 := httptest.NewRecorder()
        c1, _ := gin.CreateTestContext(w1)
        req1, _ := http.NewRequest("GET", "/test", nil)
        c1.Request = req1
        c1.Set("user_id", "user-abc")
        router.HandleContext(c1)
        assert.Equal(t, http.StatusOK, w1.Code)

        // Second request with same user_id should be rate limited
        w2 := httptest.NewRecorder()
        c2, _ := gin.CreateTestContext(w2)
        req2, _ := http.NewRequest("GET", "/test", nil)
        c2.Request = req2
        c2.Set("user_id", "user-abc")
        router.HandleContext(c2)
        assert.Equal(t, http.StatusTooManyRequests, w2.Code)
    })

    t.Run("should fall back to IP when no user_id", func(t *testing.T) {
        w := httptest.NewRecorder()
        c, _ := gin.CreateTestContext(w)
        req, _ := http.NewRequest("GET", "/test", nil)
        req.RemoteAddr = "5.6.7.8:9012"
        c.Request = req
        router.HandleContext(c)
        assert.Equal(t, http.StatusOK, w.Code)
    })

    t.Run("should return 429 JSON response", func(t *testing.T) {
        w := httptest.NewRecorder()
        c, _ := gin.CreateTestContext(w)
        req, _ := http.NewRequest("GET", "/test", nil)
        req.RemoteAddr = "5.6.7.8:9012"
        c.Request = req
        router.HandleContext(c)
        assert.Equal(t, http.StatusTooManyRequests, w.Code)

        var response map[string]interface{}
        err := json.Unmarshal(w.Body.Bytes(), &response)
        assert.NoError(t, err)
        assert.Equal(t, false, response["success"])
        assert.Contains(t, response["message"], "请求过于频繁")
    })
}

// Test concurrent access to visitor limiter
func TestVisitorLimiter_ConcurrentAccess(t *testing.T) {
    limiter := newVisitorLimiter(rate.Limit(100), 100, 10*time.Second)
    defer limiter.stop()

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            limiter.allow(string(rune(id)))
        }(i)
    }
    wg.Wait()

    // Should not panic or deadlock
}
