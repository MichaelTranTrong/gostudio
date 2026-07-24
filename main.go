package main

import (
	"gostudio/internal/auth"
	"gostudio/internal/database"
	"gostudio/internal/handlers"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Ensure upload/output directories exist
	for _, dir := range []string{"uploads", "outputs"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("cannot create dir %s: %v", dir, err)
		}
	}

	// Database
	if err := database.Connect(); err != nil {
		log.Fatalf("DB connect error: %v", err)
	}
	if err := database.Migrate(); err != nil {
		log.Fatalf("DB migrate error: %v", err)
	}

	authEnabled := auth.Enabled()
	if authEnabled {
		// Bản VPS: nối tới kho tài khoản dùng chung với Go Keep.
		if err := database.ConnectAuth(); err != nil {
			log.Fatalf("Auth DB connect error: %v", err)
		}
		log.Println("🔐 AUTH_ENABLED=true — bật đăng nhập/đăng ký (dùng chung tài khoản Go Keep)")
	} else {
		log.Println("🔓 AUTH_ENABLED tắt — chạy chế độ local, không cần đăng nhập")
	}

	r := gin.Default()

	// Serve HTML templates
	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "./web/static")

	// UI — truyền cờ authEnabled cho frontend biết có cần hiện màn đăng nhập không.
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{"authEnabled": authEnabled})
	})

	api := r.Group("/api")

	if authEnabled {
		// Bản online (VPS): KHÔNG có quay/chụp màn hình — chức năng đó chỉ chạy
		// local qua native app macOS, nên không đăng ký /api/capture/upload.

		// Route auth công khai (chưa đăng nhập vẫn gọi được).
		ag := api.Group("/auth")
		ag.POST("/register", handlers.Register)
		ag.POST("/login", handlers.Login)
		ag.POST("/forgot-password", handlers.ForgotPassword)
		ag.POST("/reset-password", handlers.ResetPassword)
		ag.GET("/verify-email", handlers.VerifyEmail)
		ag.GET("/me", auth.RequireAuth(), handlers.Me)
		ag.POST("/resend-verify", auth.RequireAuth(), handlers.ResendVerify)

		// Mọi route media yêu cầu đăng nhập.
		registerMediaRoutes(api.Group("", auth.RequireAuth()))
	} else {
		// Bản local: có quay/chụp màn hình (native app macOS dùng token riêng).
		api.POST("/capture/upload", handlers.CaptureUpload)
		registerMediaRoutes(api)
	}

	log.Println("Go Studio đang chạy tại http://localhost:2005")
	if err := r.Run(":2005"); err != nil {
		log.Fatal(err)
	}
}

// registerMediaRoutes gắn toàn bộ route xử lý media lên group g.
// Local: g = /api (không chặn). VPS: g = /api có RequireAuth.
func registerMediaRoutes(g *gin.RouterGroup) {
	g.POST("/convert/mp4-to-mp3", handlers.ConvertMP4ToMP3)
	g.POST("/trim/media", handlers.TrimMedia)
	g.POST("/concat/media", handlers.ConcatMedia)
	g.POST("/tts", handlers.TextToSpeech)
	g.GET("/voices", handlers.GetVoices)
	g.GET("/jobs", handlers.JobList)
	g.GET("/jobs/:id", handlers.JobStatus)
	g.DELETE("/jobs/:id", handlers.DeleteJob)
	g.PATCH("/jobs/:id/rename", handlers.RenameJob)
	g.DELETE("/jobs", handlers.DeleteAllJobs)
	g.GET("/download/:id", handlers.DownloadOutput)
	g.GET("/preview/:id", handlers.PreviewOutput)
}
