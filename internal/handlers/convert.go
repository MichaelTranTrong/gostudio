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

const (
	uploadDir = "uploads"
	outputDir = "outputs"
)

// ConvertMP4ToMP3 handles file upload and kicks off background conversion.
func ConvertMP4ToMP3(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng chọn file MP4"})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".mp4" && ext != ".MP4" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ file .mp4"})
		return
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

	jobID, err := models.CreateJob("mp4_to_mp3", inputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo job"})
		return
	}

	go runFFmpegConvert(jobID, inputPath, baseName)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"message": "Đang chuyển đổi, hãy kiểm tra trạng thái",
	})
}

func runFFmpegConvert(jobID uint64, inputPath, baseName string) {
	outputName := baseName + ".mp3"
	outputPath := filepath.Join(outputDir, outputName)

	_ = models.UpdateJob(jobID, "processing", "", "")

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-q:a", "2", "-map", "a", outputPath, "-y")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg error job %d: %v\n%s", jobID, err, string(out))
		_ = models.UpdateJob(jobID, "failed", "", err.Error())
		return
	}

	_ = models.UpdateJob(jobID, "done", outputPath, "")
	log.Printf("job %d done: %s", jobID, outputPath)
}

// JobStatus returns JSON status of a job.
func JobStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id không hợp lệ"})
		return
	}
	job, err := models.GetJob(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy job"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// DeleteJob xóa một job theo id.
func DeleteJob(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id không hợp lệ"})
		return
	}
	if err := models.DeleteJob(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa"})
}

// DeleteAllJobs xóa toàn bộ lịch sử và reset AUTO_INCREMENT.
func DeleteAllJobs(c *gin.Context) {
	if err := models.DeleteAllJobs(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa toàn bộ lịch sử"})
}

// JobList returns all jobs.
func JobList(c *gin.Context) {
	jobs, err := models.ListJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

// DownloadOutput serves the converted file as an attachment (tải về).
func DownloadOutput(c *gin.Context) {
	serveOutput(c, "attachment")
}

// PreviewOutput serves the converted file inline (xem/phát trực tiếp trong trình duyệt).
func PreviewOutput(c *gin.Context) {
	serveOutput(c, "inline")
}

// serveOutput resolves a done job's output file and serves it with the given
// Content-Disposition (attachment = tải về, inline = mở xem trực tiếp).
func serveOutput(c *gin.Context, disposition string) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id không hợp lệ"})
		return
	}
	job, err := models.GetJob(id)
	if err != nil || job.Status != "done" {
		c.JSON(http.StatusNotFound, gin.H{"error": "File chưa sẵn sàng hoặc không tồn tại"})
		return
	}
	if _, err := os.Stat(job.OutputFile); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File output không tìm thấy trên disk"})
		return
	}
	fileName := filepath.Base(job.OutputFile)
	encoded := url.PathEscape(fileName)
	c.Header("Content-Disposition", disposition+`; filename="`+fileName+`"; filename*=UTF-8''`+encoded)
	c.Header("Content-Type", contentTypeFor(fileName))
	c.File(job.OutputFile)
}

// contentTypeFor đoán MIME từ phần mở rộng (audio/video/ảnh).
func contentTypeFor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}
