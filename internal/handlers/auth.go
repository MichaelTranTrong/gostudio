package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"gostudio/internal/auth"
)

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Register tạo tài khoản mới trong kho tài khoản Go Keep (dùng chung),
// rồi gửi email xác thực (giống luồng Go Keep). Trả về JWT để đăng nhập luôn.
func Register(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" || req.Name == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập đầy đủ họ tên, email và mật khẩu"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mật khẩu phải có ít nhất 6 ký tự"})
		return
	}

	existing, _, err := auth.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email đã được đăng ký"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	userID, err := auth.CreateUser(req.Email, req.Name, string(hash))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}

	// Gửi email xác thực (bất đồng bộ, không chặn phản hồi).
	token := randToken()
	if err := auth.SetVerifyToken(userID, token); err == nil {
		verifyLink := auth.AppURL() + "/api/auth/verify-email?token=" + token
		go auth.SendVerifyEmail(req.Email, req.Name, verifyLink)
	}

	jwt, err := auth.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	user, _, _ := auth.GetUserByID(userID)
	c.JSON(http.StatusCreated, gin.H{"token": jwt, "user": user})
}

func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, hash, err := auth.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
		return
	}
	if user.IsLocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tài khoản đã bị khóa. Vui lòng liên hệ quản trị viên."})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
		return
	}

	jwt, err := auth.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": jwt, "user": user})
}

// Me trả về thông tin user hiện tại (dùng để hiện banner xác thực email).
func Me(c *gin.Context) {
	user, _, err := auth.GetUserByID(auth.UserID(c))
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// VerifyEmail nhận link từ email → đánh dấu đã xác thực → chuyển về trang chủ.
func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.Redirect(http.StatusSeeOther, "/?verified=invalid")
		return
	}
	if err := auth.VerifyEmailByToken(token); err != nil {
		c.Redirect(http.StatusSeeOther, "/?verified=invalid")
		return
	}
	c.Redirect(http.StatusSeeOther, "/?verified=success")
}

// ForgotPassword gửi email chứa link đặt lại mật khẩu. Luôn trả về success
// để tránh lộ email nào đã đăng ký (chống dò email).
func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	msg := gin.H{"message": "Nếu email tồn tại, link đặt lại mật khẩu đã được gửi"}

	user, _, err := auth.GetUserByEmail(req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, msg)
		return
	}
	token := randToken()
	if err := auth.SetResetToken(user.ID, token, time.Now().Add(1*time.Hour)); err == nil {
		resetLink := auth.AppURL() + "/?reset_token=" + token
		go auth.SendResetEmail(user.Email, user.Name, resetLink)
	}
	c.JSON(http.StatusOK, msg)
}

// ResetPassword đặt mật khẩu mới dựa vào token trong email.
func ResetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.Token == "" || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mật khẩu phải có ít nhất 6 ký tự"})
		return
	}
	userID, err := auth.GetUserIDByResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token không hợp lệ hoặc đã hết hạn"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	if err := auth.UpdatePassword(userID, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đặt lại mật khẩu thành công"})
}

func ResendVerify(c *gin.Context) {
	uid := auth.UserID(c)
	user, _, err := auth.GetUserByID(uid)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tài khoản"})
		return
	}
	if user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã được xác thực"})
		return
	}
	token := randToken()
	if err := auth.SetVerifyToken(uid, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi máy chủ"})
		return
	}
	verifyLink := auth.AppURL() + "/api/auth/verify-email?token=" + token
	go auth.SendVerifyEmail(user.Email, user.Name, verifyLink)
	c.JSON(http.StatusOK, gin.H{"message": "Đã gửi email xác thực"})
}
