package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// notifyURL là service gửi email dùng chung với Go Keep.
func notifyURL() string {
	if u := os.Getenv("NOTIFY_URL"); u != "" {
		return u
	}
	return "https://notify.zalink.io.vn"
}

// AppURL là địa chỉ gốc của Go Studio, dùng để dựng link xác thực trong email.
func AppURL() string {
	if u := os.Getenv("APP_URL"); u != "" {
		return u
	}
	return "http://localhost:2005"
}

// SendVerifyEmail gửi email xác thực qua notify service (giống Go Keep,
// nhưng nội dung + link do Go Studio phát hành).
func SendVerifyEmail(toEmail, toName, verifyLink string) error {
	body := map[string]any{
		"to":      toEmail,
		"subject": "Xác thực email Go Studio",
		"body": fmt.Sprintf(`Xin chào %s,

Cảm ơn bạn đã đăng ký tài khoản Go Studio!

Vui lòng xác thực địa chỉ email bằng cách nhấp vào liên kết dưới đây:

%s

Nếu bạn không tạo tài khoản này, hãy bỏ qua email này.

Go Studio Team`, toName, verifyLink),
		"isHtml": false,
	}
	payload, _ := json.Marshal(body)
	resp, err := http.Post(notifyURL()+"/api/email/send", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("notify API error: status %d", resp.StatusCode)
	}
	return nil
}
