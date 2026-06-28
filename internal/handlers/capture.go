package handlers

import (
	"fmt"
	"gostudio/internal/models"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Định dạng file capture hợp lệ từ native app macOS.
var captureExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".mp4": true, ".mov": true,
}

// CaptureUpload nhận file ảnh/video màn hình do native app macOS gửi lên.
//
// multipart form:
//   - file (bắt buộc): file capture đã encode sẵn (PNG/JPG/MP4/MOV)
//   - type (bắt buộc): "screenshot" | "screen_record"
//
// File capture chính là output cuối — native app đã quay/chụp + encode,
// nên thường lưu thẳng vào outputs/ và đánh dấu job done luôn.
// Riêng .mov sẽ transcode sang .mp4 (async) cho tương thích trình duyệt.
func CaptureUpload(c *gin.Context) {
	if !checkCaptureToken(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
		return
	}

	captureType := c.PostForm("type")
	if captureType != "screenshot" && captureType != "screen_record" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type phải là screenshot hoặc screen_record"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu file capture"})
		return
	}

	originalName, _ := url.QueryUnescape(filepath.Base(file.Filename))
	ext := strings.ToLower(filepath.Ext(originalName))
	if !captureExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng không hỗ trợ: " + ext})
		return
	}

	ts := time.Now().UnixMilli()
	storedName := fmt.Sprintf("%d_%s", ts, originalName)
	storedPath := filepath.Join(outputDir, storedName)
	if err := c.SaveUploadedFile(file, storedPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file capture"})
		return
	}

	jobID, err := models.CreateJob(captureType, captureLabel(captureType, originalName))
	if err != nil {
		os.Remove(storedPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo job"})
		return
	}

	// Cần xử lý FFmpeg nếu: file .mov (transcode) hoặc có vùng cắt (crop).
	cropFilter := buildCropFilter(c)
	if ext == ".mov" || cropFilter != "" {
		go processCaptureVideo(jobID, storedPath, ts, cropFilter)
	} else {
		_ = models.UpdateJob(jobID, "done", storedPath, "")
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "message": "Đã nhận file capture"})
}

// processCaptureVideo chạy FFmpeg: transcode .mov→.mp4 và/hoặc cắt vùng (crop).
func processCaptureVideo(jobID uint64, inputPath string, ts int64, cropFilter string) {
	_ = models.UpdateJob(jobID, "processing", "", "")

	mp4Path := filepath.Join(outputDir, fmt.Sprintf("%d_screen.mp4", ts))
	args := []string{"-i", inputPath}
	if cropFilter != "" {
		args = append(args, "-vf", cropFilter)
	}
	args = append(args, "-c:v", "libx264", "-preset", "fast", "-c:a", "aac", mp4Path, "-y")

	out, err := exec.Command("ffmpeg", args...).CombinedOutput()
	if err != nil {
		log.Printf("capture video job %d: %v\n%s", jobID, err, string(out))
		_ = models.UpdateJob(jobID, "failed", "", err.Error())
		return
	}

	os.Remove(inputPath)
	_ = models.UpdateJob(jobID, "done", mp4Path, "")
	log.Printf("capture job %d done: %s", jobID, mp4Path)
}

// buildCropFilter trả "crop=w:h:x:y" nếu có đủ 4 tham số nguyên hợp lệ, ngược lại "".
func buildCropFilter(c *gin.Context) string {
	w, h := c.PostForm("crop_w"), c.PostForm("crop_h")
	x, y := c.PostForm("crop_x"), c.PostForm("crop_y")
	for _, v := range []string{w, h, x, y} {
		if v == "" {
			return ""
		}
		if _, err := strconv.Atoi(v); err != nil {
			return ""
		}
	}
	return fmt.Sprintf("crop=%s:%s:%s:%s", w, h, x, y)
}

func captureLabel(captureType, name string) string {
	if captureType == "screenshot" {
		return "[Ảnh] " + name
	}
	return "[Quay] " + name
}

// checkCaptureToken cho qua nếu không đặt CAPTURE_TOKEN; nếu có thì so khớp
// header X-Capture-Token (hoặc field token trong form).
func checkCaptureToken(c *gin.Context) bool {
	want := os.Getenv("CAPTURE_TOKEN")
	if want == "" {
		return true
	}
	got := c.GetHeader("X-Capture-Token")
	if got == "" {
		got = c.PostForm("token")
	}
	return got == want
}
