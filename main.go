package main

import (
	"net/http"
	"strings"

	"github.com/dogslee/youtube_transcript_api"
	"github.com/gin-gonic/gin"
)

type TranscriptResponse struct {
	VideoID    string                      `json:"video_id"`
	Language   string                      `json:"language"`
	IsGenerated bool                       `json:"is_generated"`
	Snippets   []youtube_transcript_api.FetchedTranscriptSnippet `json:"snippets"`
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
			text.WriteString(snippet.Text + " ")
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
		VideoID:      transcript.VideoID,
		Language:     transcript.Language,
		IsGenerated:  transcript.IsGenerated,
		Snippets:     transcript.Snippets,
	}
	c.JSON(http.StatusOK, resp)
}

func main() {
	r := gin.Default()
	r.GET("/transcript/:id", getTranscript)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.Run(":8080")
}