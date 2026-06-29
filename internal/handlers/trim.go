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

// videoExts / audioExts là các định dạng được chấp nhận để cắt.
var videoExts = map[string]bool{
	".mp4": true, ".mov": true, ".mkv": true, ".webm": true, ".avi": true, ".m4v": true,
}
var audioExts = map[string]bool{
	".mp3": true, ".wav": true, ".m4a": true, ".aac": true, ".flac": true, ".ogg": true,
}

// TrimMedia cắt một đoạn video/audio theo thời gian (chạy FFmpeg async).
// Nguồn có thể là file upload mới (field "file") HOẶC một job có sẵn (field "source_id").
func TrimMedia(c *gin.Context) {
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

	var duration float64 // 0 = cắt tới hết
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

	var inputPath, ext, baseName, displayInput string

	if sourceID := strings.TrimSpace(c.PostForm("source_id")); sourceID != "" {
		// Nguồn là một job có sẵn → dùng thẳng file output của nó làm input.
		id, err := strconv.ParseUint(sourceID, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_id không hợp lệ"})
			return
		}
		src, err := models.GetJob(id)
		if err != nil || src.Status != "done" || src.OutputFile == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "File nguồn chưa sẵn sàng"})
			return
		}
		if _, err := os.Stat(src.OutputFile); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File nguồn không còn trên disk"})
			return
		}
		inputPath = src.OutputFile
		ext = strings.ToLower(filepath.Ext(inputPath))
		if !videoExts[ext] && !audioExts[ext] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng nguồn không cắt được"})
			return
		}
		base := filepath.Base(inputPath)
		baseName = strings.TrimSuffix(base, filepath.Ext(base))
		// Nhãn (không phải path thật) → xóa job cắt không đụng vào file nguồn.
		displayInput = "[Cắt] " + base
	} else {
		// Nguồn là file upload mới.
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng chọn file video hoặc audio"})
			return
		}
		ext = strings.ToLower(filepath.Ext(file.Filename))
		if !videoExts[ext] && !audioExts[ext] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng không hỗ trợ"})
			return
		}
		ts := time.Now().UnixMilli()
		originalName, _ := url.QueryUnescape(filepath.Base(file.Filename))
		baseName = strings.TrimSuffix(originalName, filepath.Ext(originalName))
		inputName := fmt.Sprintf("%d_%s", ts, filepath.Base(file.Filename))
		inputPath = filepath.Join(uploadDir, inputName)
		if err := c.SaveUploadedFile(file, inputPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file upload"})
			return
		}
		displayInput = inputPath // path thật → xóa job dọn được file gốc
	}

	jobType := "video_trim"
	if audioExts[ext] {
		jobType = "audio_trim"
	}
	jobID, err := models.CreateJob(jobType, displayInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo job"})
		return
	}

	go runTrim(jobID, inputPath, baseName, ext, startSec, duration)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"message": "Đang cắt, hãy kiểm tra trạng thái",
	})
}

func runTrim(jobID uint64, inputPath, baseName, srcExt string, startSec, duration float64) {
	outExt, encodeArgs := trimEncodeArgs(srcExt)
	outputPath := uniqueOutputPath(baseName+"_cat", outExt)

	_ = models.UpdateJob(jobID, "processing", "", "")

	// -ss trước -i: seek nhanh; re-encode đảm bảo cắt đúng frame.
	args := []string{"-ss", strconv.FormatFloat(startSec, 'f', 3, 64), "-i", inputPath}
	if duration > 0 {
		args = append(args, "-t", strconv.FormatFloat(duration, 'f', 3, 64))
	}
	args = append(args, encodeArgs...)
	args = append(args, outputPath, "-y")

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

// uniqueOutputPath trả về đường dẫn chưa tồn tại trong outputDir (tránh ghi đè khi
// cắt nhiều lần cùng một file): base.ext, rồi base-2.ext, base-3.ext, …
func uniqueOutputPath(base, ext string) string {
	p := filepath.Join(outputDir, base+ext)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	for i := 2; ; i++ {
		p = filepath.Join(outputDir, fmt.Sprintf("%s-%d%s", base, i, ext))
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

// trimEncodeArgs chọn đuôi file output + tham số encode theo định dạng nguồn.
func trimEncodeArgs(srcExt string) (string, []string) {
	switch srcExt {
	case ".mp3":
		return ".mp3", []string{"-c:a", "libmp3lame", "-q:a", "2"}
	case ".wav":
		return ".wav", []string{"-c:a", "pcm_s16le"}
	case ".flac":
		return ".flac", []string{"-c:a", "flac"}
	case ".ogg":
		return ".ogg", []string{"-c:a", "libvorbis"}
	case ".m4a", ".aac":
		return ".m4a", []string{"-c:a", "aac"}
	default: // video → MP4 (H.264/AAC)
		return ".mp4", []string{
			"-c:v", "libx264", "-preset", "veryfast",
			"-c:a", "aac",
			"-movflags", "+faststart",
		}
	}
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
