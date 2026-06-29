package handlers

import (
	"fmt"
	"gostudio/internal/models"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// videoExts là các định dạng video được chấp nhận để cắt.
var videoExts = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true, ".webm": true, ".avi": true, ".m4v": true,
}

// TrimVideo cắt một đoạn video theo thời gian bắt đầu/kết thúc (chạy FFmpeg async).
func TrimVideo(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng chọn file video"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !videoExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng video không hỗ trợ"})
		return
	}

	startStr := strings.TrimSpace(c.PostForm("start"))
	endStr := strings.TrimSpace(c.PostForm("end"))
	if startStr == "" {
		startStr = "0"
	}

	startSec, ok := parseTimecode(startStr)
	if !ok || startSec < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian bắt đầu không hợp lệ (giây hoặc HH:MM:SS)"})
		return
	}

	var duration float64 // 0 = cắt tới hết video
	if endStr != "" {
		endSec, ok := parseTimecode(endStr)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian kết thúc không hợp lệ (giây hoặc HH:MM:SS)"})
			return
		}
		if endSec <= startSec {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian kết thúc phải lớn hơn thời gian bắt đầu"})
			return
		}
		duration = endSec - startSec
	}

	ts := time.Now().UnixMilli()
	originalName, _ := url.QueryUnescape(filepath.Base(file.Filename))
	baseName := strings.TrimSuffix(originalName, filepath.Ext(originalName))
	inputName := fmt.Sprintf("%d_%s", ts, filepath.Base(file.Filename))
	inputPath := filepath.Join(uploadDir, inputName)

	if err := c.SaveUploadedFile(file, inputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file upload"})
		return
	}

	jobID, err := models.CreateJob("video_trim", inputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo job"})
		return
	}

	go runTrim(jobID, inputPath, baseName, startSec, duration)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"message": "Đang cắt video, hãy kiểm tra trạng thái",
	})
}

func runTrim(jobID uint64, inputPath, baseName string, startSec, duration float64) {
	outputName := baseName + "_cat.mp4"
	outputPath := filepath.Join(outputDir, outputName)

	_ = models.UpdateJob(jobID, "processing", "", "")

	// -ss trước -i: seek nhanh; re-encode đảm bảo cắt đúng frame.
	args := []string{"-ss", strconv.FormatFloat(startSec, 'f', 3, 64), "-i", inputPath}
	if duration > 0 {
		args = append(args, "-t", strconv.FormatFloat(duration, 'f', 3, 64))
	}
	args = append(args,
		"-c:v", "libx264", "-preset", "veryfast",
		"-c:a", "aac",
		"-movflags", "+faststart",
		outputPath, "-y",
	)

	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg trim error job %d: %v\n%s", jobID, err, string(out))
		_ = models.UpdateJob(jobID, "failed", "", err.Error())
		return
	}

	_ = models.UpdateJob(jobID, "done", outputPath, "")
	log.Printf("job %d trim done: %s", jobID, outputPath)
}

// parseTimecode chấp nhận "SS", "MM:SS", "HH:MM:SS" (cho phép phần thập phân) → tổng số giây.
func parseTimecode(s string) (float64, bool) {
	parts := strings.Split(s, ":")
	if len(parts) > 3 {
		return 0, false
	}
	var total float64
	for _, p := range parts {
		if p == "" {
			return 0, false
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil || v < 0 {
			return 0, false
		}
		total = total*60 + v
	}
	return total, true
}
