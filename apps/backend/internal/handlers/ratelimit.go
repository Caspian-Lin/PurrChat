package handlers

import (
    "net/http"
    "sync"
    "time"

    "purr-chat-server/pkg/logger"

    "github.com/gin-gonic/gin"
    "golang.org/x/time/rate"
)

// visitor 跟踪单个客户端（IP 或用户）的速率限制状态
type visitor struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

// visitorLimiter 管理 per-key 速率限制，支持自动清理过期条目
type visitorLimiter struct {
    mu       sync.Mutex
    visitors map[string]*visitor
    rate     rate.Limit
    burst    int
    ttl      time.Duration
    done     chan struct{}
}

// newVisitorLimiter 创建一个新的 per-key 速率限制器
// r: 每秒补充的令牌数（rate.Limit 类型）
// burst: 令牌桶最大容量（允许的突发请求数）
// ttl: 条目过期时间（超过此时间未活动的客户端将被清理）
func newVisitorLimiter(r rate.Limit, burst int, ttl time.Duration) *visitorLimiter {
    vl := &visitorLimiter{
        visitors: make(map[string]*visitor),
        rate:     r,
        burst:    burst,
        ttl:      ttl,
        done:     make(chan struct{}),
    }

    // 启动后台清理协程，定期移除不活跃的条目防止内存泄漏
    go vl.cleanup()

    return vl
}

// getLimiter 获取或创建指定 key 的令牌桶限流器
func (vl *visitorLimiter) getLimiter(key string) *rate.Limiter {
    vl.mu.Lock()
    defer vl.mu.Unlock()

    v, exists := vl.visitors[key]
    if !exists {
        v = &visitor{
            limiter:  rate.NewLimiter(vl.rate, vl.burst),
            lastSeen: time.Now(),
        }
        vl.visitors[key] = v
    }

    v.lastSeen = time.Now()
    return v.limiter
}

// allow 检查指定 key 是否允许请求
func (vl *visitorLimiter) allow(key string) bool {
    return vl.getLimiter(key).Allow()
}

// cleanup 定期清理过期的 visitor 条目
func (vl *visitorLimiter) cleanup() {
    ticker := time.NewTicker(vl.ttl / 2)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            vl.mu.Lock()
            now := time.Now()
            for key, v := range vl.visitors {
                if now.Sub(v.lastSeen) > vl.ttl {
                    delete(vl.visitors, key)
                }
            }
            vl.mu.Unlock()
        case <-vl.done:
            return
        }
    }
}

// stop 停止清理协程（用于优雅关闭）
func (vl *visitorLimiter) stop() {
    close(vl.done)
}

// ==================== Gin 中间件工厂函数 ====================

// IPRateLimitMiddleware 创建基于客户端 IP 的速率限制中间件
// 适用于全局限流和未认证端点
func IPRateLimitMiddleware(r rate.Limit, burst int, ttl time.Duration) gin.HandlerFunc {
    limiter := newVisitorLimiter(r, burst, ttl)

    return func(c *gin.Context) {
        ip := c.ClientIP()
        if !limiter.allow(ip) {
            logger.InfofWithCaller("Rate limit exceeded for IP: %s on %s %s", ip, c.Request.Method, c.Request.URL.Path)
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "success": false,
                "message": "请求过于频繁，请稍后再试",
            })
            return
        }
        c.Next()
    }
}

// UserRateLimitMiddleware 创建基于用户 ID 的速率限制中间件
// 必须在 AuthMiddleware 之后使用（依赖 c.Get("user_id")）
// 退回到 IP 限流（当用户未认证时）
func UserRateLimitMiddleware(r rate.Limit, burst int, ttl time.Duration) gin.HandlerFunc {
    limiter := newVisitorLimiter(r, burst, ttl)

    return func(c *gin.Context) {
        // 优先使用 user_id，退回到 IP
        key := c.ClientIP()
        if userID, exists := c.Get("user_id"); exists {
            key = userID.(string)
        }

        if !limiter.allow(key) {
            logger.InfofWithCaller("Rate limit exceeded for user/IP: %s on %s %s", key, c.Request.Method, c.Request.URL.Path)
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "success": false,
                "message": "请求过于频繁，请稍后再试",
            })
            return
        }
        c.Next()
    }
}
