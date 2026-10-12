package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dogslee/youtube_transcript_api"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type TranscriptResponse struct {
	VideoID     string                                            `json:"video_id"`
	Language    string                                            `json:"language"`
	IsGenerated bool                                              `json:"is_generated"`
	Snippets    []youtube_transcript_api.FetchedTranscriptSnippet `json:"snippets"`
}

type FullTextResponse struct {
	VideoID string `json:"video_id"`
	Title   string `json:"title"`
	Text    string `json:"fulltext"`
}

func extractVideoID(input string) string {
	return strings.TrimSpace(input)
}

func getTranscript(c *gin.Context) {
	videoID := extractVideoID(c.Param("id"))
	if len(videoID) != 11 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid YouTube video ID"})
		return
	}

	fullText := c.DefaultQuery("fulltext", "false") == "true"
	lang := c.DefaultQuery("lang", "en")
	languages := []string{lang}

	api, err := youtube_transcript_api.NewYouTubeTranscriptApi(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API client"})
		return
	}

	transcript, err := api.Fetch(videoID, languages, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if fullText {
		var text strings.Builder
		for _, snippet := range transcript.Snippets {
			text.WriteString(snippet.Text)
			text.WriteString(" ")
		}
		resp := FullTextResponse{
			VideoID: transcript.VideoID,
			Title:   transcript.Title,
			Text:    strings.TrimSpace(text.String()),
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	// Original structured response
	resp := TranscriptResponse{
		VideoID:     transcript.VideoID,
		Language:    transcript.Language,
		IsGenerated: transcript.IsGenerated,
		Snippets:    transcript.Snippets,
	}
	c.JSON(http.StatusOK, resp)
}

func getPort() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9898"
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("PORT must be a number between 1 and 65535, got %q", port)
	}

	return port, nil
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		panic(fmt.Errorf("load .env: %w", err))
	}

	gin.SetMode(gin.ReleaseMode)

	port, err := getPort()
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(fmt.Errorf("configure trusted proxies: %w", err))
	}
	r.GET("/transcript/:id", getTranscript)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	if err := r.Run(":" + port); err != nil {
		panic(fmt.Errorf("start server: %w", err))
	}
}
