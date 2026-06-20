package main

import (
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

	r := gin.Default()

	// Serve HTML templates
	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "./web/static")

	// UI
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// API
	api := r.Group("/api")
	{
		api.POST("/convert/mp4-to-mp3", handlers.ConvertMP4ToMP3)
		api.GET("/jobs", handlers.JobList)
		api.GET("/jobs/:id", handlers.JobStatus)
		api.DELETE("/jobs/:id", handlers.DeleteJob)
		api.DELETE("/jobs", handlers.DeleteAllJobs)
		api.GET("/download/:id", handlers.DownloadOutput)
	}

	log.Println("Go Studio đang chạy tại http://localhost:2005")
	if err := r.Run(":2005"); err != nil {
		log.Fatal(err)
	}
}
