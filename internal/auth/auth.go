// Package auth xử lý đăng ký / đăng nhập dùng chung tài khoản với Go Keep.
//
// Chỉ hoạt động khi AUTH_ENABLED=true (bản VPS). Khi tắt (bản local),
// toàn bộ package này không được nối vào router — app chạy như cũ, không cần đăng nhập.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const ctxUserIDKey = "authUserID"

// Enabled cho biết tính năng tài khoản có bật không (feature flag).
func Enabled() bool {
	return os.Getenv("AUTH_ENABLED") == "true"
}

func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("gostudio-dev-secret-change-me")
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func sign(input string) string {
	m := hmac.New(sha256.New, jwtSecret())
	m.Write([]byte(input))
	return b64(m.Sum(nil))
}

// GenerateToken tạo JWT HS256 (tự cài bằng stdlib, không cần thư viện ngoài).
// Token này là session RIÊNG của Go Studio — độc lập với token của Go Keep,
// dù hai bên dùng chung bảng users.
func GenerateToken(userID uint64) (string, error) {
	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	signingInput := header + "." + b64(payload)
	return signingInput + "." + sign(signingInput), nil
}

// ParseToken kiểm tra chữ ký + hạn dùng, trả về user id.
func ParseToken(token string) (uint64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("token không hợp lệ")
	}
	if !hmac.Equal([]byte(parts[2]), []byte(sign(parts[0]+"."+parts[1]))) {
		return 0, errors.New("chữ ký không hợp lệ")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}
	var claims struct {
		Sub uint64 `json:"sub"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return 0, err
	}
	if claims.Sub == 0 {
		return 0, errors.New("token thiếu sub")
	}
	if claims.Exp < time.Now().Unix() {
		return 0, errors.New("token đã hết hạn")
	}
	return claims.Sub, nil
}

// extractToken lấy token từ header Authorization hoặc cookie.
// Cookie cần cho các thẻ <img>/<video>/<a download> (không gắn header được).
func extractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if ck, err := c.Cookie("gostudio_token"); err == nil {
		return ck
	}
	return ""
}

// RequireAuth chặn request nếu chưa đăng nhập (gắn vào các route media khi AUTH bật).
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, err := ParseToken(extractToken(c))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Vui lòng đăng nhập"})
			return
		}
		c.Set(ctxUserIDKey, uid)
		c.Next()
	}
}

// UserID trả về id user hiện tại. Trả 0 khi AUTH tắt hoặc route không qua RequireAuth
// (vd endpoint capture của native app) → model sẽ không lọc theo user.
func UserID(c *gin.Context) uint64 {
	if v, ok := c.Get(ctxUserIDKey); ok {
		if id, ok := v.(uint64); ok {
			return id
		}
	}
	return 0
}
