package handlers

import (
	"encoding/json"
	"fmt"
	"gostudio/internal/models"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// concatOrderItem mô tả 1 phần tử trong thứ tự ghép: hoặc "upload" (lấy file kế
// tiếp trong field "files") hoặc "source" (job có sẵn trong lịch sử, dùng OutputFile).
type concatOrderItem struct {
	Type string `json:"type"`
	ID   uint64 `json:"id,omitempty"`
}

type concatInput struct {
	Path string
	Ext  string
	Temp bool // true = file upload tạm, xóa sau khi ghép xong (không phải job khác đang giữ)
}

// ConcatMedia ghép nhiều video (hoặc nhiều audio) thành 1 file theo đúng thứ tự đã chọn.
func ConcatMedia(c *gin.Context) {
	var order []concatOrderItem
	if err := json.Unmarshal([]byte(c.PostForm("order")), &order); err != nil || len(order) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần ít nhất 2 file để ghép"})
		return
	}

	form, _ := c.MultipartForm()
	var uploadFiles []*multipart.FileHeader
	if form != nil {
		uploadFiles = form.File["files"]
	}

	// Bước 1: xác định nguồn + đuôi file + kiểm tra loại (video/audio) khớp nhau cho
	// TẤT CẢ phần tử TRƯỚC khi lưu bất kỳ file upload nào — tránh rác file trên disk
	// nếu request bị từ chối giữa chừng (vd ghép lẫn video+audio).
	type resolved struct {
		fh   *multipart.FileHeader // nil nếu là "source"
		path string                // đã có sẵn nếu là "source"
		ext  string
	}
	resolvedItems := make([]resolved, 0, len(order))
	uploadIdx := 0
	var kind, firstBaseName string

	for i, item := range order {
		var path, ext string
		var fh *multipart.FileHeader

		if item.Type == "source" {
			src, err := models.GetJob(item.ID)
			if err != nil || src.Status != "done" || src.OutputFile == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("File nguồn #%d chưa sẵn sàng", i+1)})
				return
			}
			if _, err := os.Stat(src.OutputFile); os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("File nguồn #%d không còn trên disk", i+1)})
				return
			}
			path = src.OutputFile
			ext = strings.ToLower(filepath.Ext(path))
		} else {
			if uploadIdx >= len(uploadFiles) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu file upload"})
				return
			}
			fh = uploadFiles[uploadIdx]
			uploadIdx++
			ext = strings.ToLower(filepath.Ext(fh.Filename))
		}

		if !videoExts[ext] && !audioExts[ext] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng không hỗ trợ: " + ext})
			return
		}
		itemKind := "audio"
		if videoExts[ext] {
			itemKind = "video"
		}
		if kind == "" {
			kind = itemKind
			base := filepath.Base(path)
			if fh != nil {
				base = filepath.Base(fh.Filename)
			}
			firstBaseName = strings.TrimSuffix(base, filepath.Ext(base))
		} else if kind != itemKind {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể ghép lẫn video và audio trong cùng 1 lần"})
			return
		}

		resolvedItems = append(resolvedItems, resolved{fh: fh, path: path, ext: ext})
	}

	// Bước 2: mọi kiểm tra đã qua → giờ mới lưu các file upload xuống disk.
	inputs := make([]concatInput, 0, len(resolvedItems))
	for i, r := range resolvedItems {
		if r.fh == nil {
			inputs = append(inputs, concatInput{Path: r.path, Ext: r.ext})
			continue
		}
		ts := time.Now().UnixMilli()
		savePath := filepath.Join(uploadDir, fmt.Sprintf("%d_%d_%s", ts, i, filepath.Base(r.fh.Filename)))
		if err := c.SaveUploadedFile(r.fh, savePath); err != nil {
			for _, in := range inputs {
				if in.Temp {
					os.Remove(in.Path)
				}
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file upload"})
			return
		}
		inputs = append(inputs, concatInput{Path: savePath, Ext: r.ext, Temp: true})
	}

	jobType := "video_concat"
	if kind == "audio" {
		jobType = "audio_concat"
	}
	jobID, err := models.CreateJob(jobType, fmt.Sprintf("[Ghép] %d file", len(inputs)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo job"})
		return
	}

	go runConcat(jobID, inputs, kind, firstBaseName)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  jobID,
		"message": "Đang ghép, hãy kiểm tra trạng thái",
	})
}

func runConcat(jobID uint64, inputs []concatInput, kind, baseName string) {
	defer func() {
		for _, in := range inputs {
			if in.Temp {
				os.Remove(in.Path)
			}
		}
	}()

	_ = models.UpdateJob(jobID, "processing", "", "")

	args := make([]string, 0, len(inputs)*2+10)
	for _, in := range inputs {
		args = append(args, "-i", in.Path)
	}

	var outputPath string
	var filter strings.Builder

	if kind == "video" {
		w, h, fps := probeVideoDims(inputs[0].Path)
		for i := range inputs {
			fmt.Fprintf(&filter, "[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=%s[v%d];", i, w, h, w, h, fps, i)
			fmt.Fprintf(&filter, "[%d:a]aformat=sample_rates=44100:channel_layouts=stereo[a%d];", i, i)
		}
		for i := range inputs {
			fmt.Fprintf(&filter, "[v%d][a%d]", i, i)
		}
		fmt.Fprintf(&filter, "concat=n=%d:v=1:a=1[outv][outa]", len(inputs))

		outputPath = uniqueOutputPath(baseName+"_ghep", ".mp4")
		args = append(args,
			"-filter_complex", filter.String(),
			"-map", "[outv]", "-map", "[outa]",
			"-c:v", "libx264", "-preset", "veryfast",
			"-c:a", "aac",
			"-movflags", "+faststart",
		)
	} else {
		outExt, encodeArgs := trimEncodeArgs(inputs[0].Ext)
		for i := range inputs {
			fmt.Fprintf(&filter, "[%d:a]aformat=sample_rates=44100:channel_layouts=stereo[a%d];", i, i)
		}
		for i := range inputs {
			fmt.Fprintf(&filter, "[a%d]", i)
		}
		fmt.Fprintf(&filter, "concat=n=%d:v=0:a=1[outa]", len(inputs))

		outputPath = uniqueOutputPath(baseName+"_ghep", outExt)
		args = append(args, "-filter_complex", filter.String(), "-map", "[outa]")
		args = append(args, encodeArgs...)
	}
	args = append(args, outputPath, "-y")

	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg concat error job %d: %v\n%s", jobID, err, string(out))
		_ = models.UpdateJob(jobID, "failed", "", err.Error())
		return
	}

	_ = models.UpdateJob(jobID, "done", outputPath, "")
	log.Printf("job %d concat done: %s", jobID, outputPath)
}

// probeVideoDims đọc độ phân giải + fps của video đầu tiên (làm chuẩn để scale/pad
// các video còn lại), rounding về số chẵn (libx264 yêu cầu). Lỗi → mặc định 1280x720@30.
func probeVideoDims(path string) (int, int, string) {
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate", "-of", "csv=p=0", path).Output()
	if err != nil {
		return 1280, 720, "30"
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) < 3 {
		return 1280, 720, "30"
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 1280, 720, "30"
	}
	if w%2 != 0 {
		w--
	}
	if h%2 != 0 {
		h--
	}
	return w, h, parts[2]
}
