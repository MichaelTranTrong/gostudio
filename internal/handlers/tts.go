package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gostudio/internal/models"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func vieneuURL() string {
	if u := os.Getenv("VIENEU_URL"); u != "" {
		return u
	}
	return "http://vieneu:8001"
}

func TextToSpeech(c *gin.Context) {
	text := strings.TrimSpace(c.PostForm("text"))
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập văn bản"})
		return
	}
	voiceID := c.PostForm("voice_id")

	displayText := string([]rune(text))
	if len([]rune(displayText)) > 80 {
		displayText = string([]rune(displayText)[:80]) + "..."
	}

	jobID, err := models.CreateJob("text_to_speech", "[TTS] "+displayText)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo job"})
		return
	}

	go runTTS(jobID, text, voiceID)

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID, "message": "Đang xử lý TTS"})
}

func GetVoices(c *gin.Context) {
	resp, err := http.Get(vieneuURL() + "/voices")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "VieNeu chưa sẵn sàng"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

func runTTS(jobID uint64, text, voiceID string) {
	_ = models.UpdateJob(jobID, "processing", "", "")

	chunks := splitText(text, 2800)
	ts := time.Now().UnixMilli()
	client := &http.Client{Timeout: 120 * time.Second}

	var wavFiles []string
	defer func() {
		for _, f := range wavFiles {
			os.Remove(f)
		}
	}()

	for i, chunk := range chunks {
		payload, _ := json.Marshal(map[string]string{
			"text":     chunk,
			"voice_id": voiceID,
		})
		resp, err := client.Post(vieneuURL()+"/stream", "application/json", bytes.NewReader(payload))
		if err != nil {
			log.Printf("TTS job %d chunk %d: %v", jobID, i, err)
			_ = models.UpdateJob(jobID, "failed", "", "Không kết nối được VieNeu: "+err.Error())
			return
		}
		wavBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 {
			_ = models.UpdateJob(jobID, "failed", "", fmt.Sprintf("VieNeu lỗi chunk %d (status %d)", i, resp.StatusCode))
			return
		}

		wavPath := filepath.Join(outputDir, fmt.Sprintf("%d_chunk%d.wav", ts, i))
		if err := os.WriteFile(wavPath, wavBytes, 0644); err != nil {
			_ = models.UpdateJob(jobID, "failed", "", err.Error())
			return
		}
		wavFiles = append(wavFiles, wavPath)
	}

	outputPath := filepath.Join(outputDir, fmt.Sprintf("%d_tts.mp3", ts))
	if err := mergeWAVtoMP3(wavFiles, outputPath); err != nil {
		log.Printf("TTS merge job %d: %v", jobID, err)
		_ = models.UpdateJob(jobID, "failed", "", err.Error())
		return
	}

	_ = models.UpdateJob(jobID, "done", outputPath, "")
	log.Printf("TTS job %d done: %s", jobID, outputPath)
}

func mergeWAVtoMP3(wavFiles []string, outputPath string) error {
	if len(wavFiles) == 1 {
		out, err := exec.Command("ffmpeg", "-i", wavFiles[0], "-q:a", "2", outputPath, "-y").CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %s", err, string(out))
		}
		return nil
	}

	listPath := outputPath + ".txt"
	var sb strings.Builder
	for _, f := range wavFiles {
		sb.WriteString("file '" + f + "'\n")
	}
	if err := os.WriteFile(listPath, []byte(sb.String()), 0644); err != nil {
		return err
	}
	defer os.Remove(listPath)

	out, err := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listPath, "-q:a", "2", outputPath, "-y").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}

func splitText(text string, maxChars int) []string {
	var chunks []string
	runes := []rune(strings.TrimSpace(text))
	for len(runes) > 0 {
		if len(runes) <= maxChars {
			chunks = append(chunks, string(runes))
			break
		}
		window := string(runes[:maxChars])
		splitAt := strings.LastIndexAny(window, ".?!\n")
		if splitAt < 0 {
			splitAt = strings.LastIndex(window, " ")
		}
		if splitAt < 0 {
			splitAt = maxChars
		} else {
			splitAt++
		}
		chunks = append(chunks, strings.TrimSpace(string(runes[:splitAt])))
		runes = []rune(strings.TrimSpace(string(runes[splitAt:])))
	}
	return chunks
}
