package main

import (
	"errors"
	"net/http"
	"os"
	"strconv"
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

const defaultLanguage = "en"

func parseLanguageCodes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{defaultLanguage}
	}

	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	languages := make([]string, 0, len(parts))

	for _, part := range parts {
		code := strings.TrimSpace(part)
		if code == "" {
			continue
		}
		dedupeKey := strings.ToLower(code)
		if _, ok := seen[dedupeKey]; ok {
			continue
		}
		seen[dedupeKey] = struct{}{}
		languages = append(languages, code)
	}

	if len(languages) == 0 {
		return []string{defaultLanguage}
	}

	return languages
}

func parseBoolQuery(c *gin.Context, key string, defaultValue bool) bool {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultValue
	}

	return value
}

func normalizeErrorText(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}

	msg = strings.Join(strings.Fields(msg), " ")
	if len(msg) > 320 {
		return msg[:320] + "..."
	}

	return msg
}

func extractCause(err error) string {
	var withCause interface{ Cause() string }
	if errors.As(err, &withCause) {
		return normalizeErrorText(withCause.Cause())
	}

	return ""
}

func buildTranscriptAPI() (*youtube_transcript_api.YouTubeTranscriptApi, error) {
	httpProxy := strings.TrimSpace(os.Getenv("YT_PROXY_HTTP"))
	httpsProxy := strings.TrimSpace(os.Getenv("YT_PROXY_HTTPS"))

	if httpProxy == "" && httpsProxy == "" {
		return youtube_transcript_api.NewYouTubeTranscriptApi(nil)
	}

	proxyConfig, err := youtube_transcript_api.NewGenericProxyConfig(httpProxy, httpsProxy)
	if err != nil {
		return nil, err
	}

	return youtube_transcript_api.NewYouTubeTranscriptApi(proxyConfig)
}

func mapTranscriptError(err error) (int, gin.H) {
	if err == nil {
		return http.StatusBadGateway, gin.H{"error": "Failed to fetch transcript from YouTube"}
	}

	var invalidVideoID *youtube_transcript_api.InvalidVideoId
	if errors.As(err, &invalidVideoID) {
		return http.StatusBadRequest, gin.H{"error": "Invalid YouTube video ID"}
	}

	var videoUnavailable *youtube_transcript_api.VideoUnavailable
	if errors.As(err, &videoUnavailable) {
		return http.StatusNotFound, gin.H{"error": "Video is unavailable"}
	}

	var transcriptsDisabled *youtube_transcript_api.TranscriptsDisabled
	if errors.As(err, &transcriptsDisabled) {
		return http.StatusNotFound, gin.H{"error": "Transcripts are disabled for this video"}
	}

	var noTranscriptFound *youtube_transcript_api.NoTranscriptFound
	if errors.As(err, &noTranscriptFound) {
		payload := gin.H{"error": "No transcript found for the requested language(s)"}
		if cause := extractCause(err); cause != "" {
			payload["cause"] = cause
		}
		return http.StatusNotFound, payload
	}

	var ageRestricted *youtube_transcript_api.AgeRestricted
	if errors.As(err, &ageRestricted) {
		payload := gin.H{"error": "This video is age-restricted"}
		if cause := extractCause(err); cause != "" {
			payload["cause"] = cause
		}
		return http.StatusForbidden, payload
	}

	var requestBlocked *youtube_transcript_api.RequestBlocked
	var ipBlocked *youtube_transcript_api.IpBlocked
	if errors.As(err, &requestBlocked) || errors.As(err, &ipBlocked) {
		payload := gin.H{
			"error": "YouTube blocked requests from this server IP",
			"hint":  "Configure YT_PROXY_HTTP and/or YT_PROXY_HTTPS with a residential rotating proxy, then retry",
		}
		if cause := extractCause(err); cause != "" {
			payload["cause"] = cause
		}
		return http.StatusTooManyRequests, payload
	}

	var poTokenRequired *youtube_transcript_api.PoTokenRequired
	if errors.As(err, &poTokenRequired) {
		payload := gin.H{"error": "YouTube requires a PO token for this video"}
		if cause := extractCause(err); cause != "" {
			payload["cause"] = cause
		}
		return http.StatusBadGateway, payload
	}

	var requestFailed *youtube_transcript_api.YouTubeRequestFailed
	if errors.As(err, &requestFailed) {
		payload := gin.H{"error": "Failed to fetch transcript from YouTube"}
		if cause := extractCause(err); cause != "" {
			payload["cause"] = cause
		}
		return http.StatusBadGateway, payload
	}

	payload := gin.H{"error": "Could not retrieve transcript"}
	if cause := extractCause(err); cause != "" {
		payload["cause"] = cause
	} else {
		payload["cause"] = normalizeErrorText(err.Error())
	}

	return http.StatusBadGateway, payload
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

	fullText := parseBoolQuery(c, "fulltext", false)
	languages := parseLanguageCodes(c.DefaultQuery("lang", defaultLanguage))

	api, err := buildTranscriptAPI()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create API client",
			"cause": normalizeErrorText(err.Error()),
		})
		return
	}

	transcript, err := api.Fetch(videoID, languages, false)
	if err != nil {
		status, payload := mapTranscriptError(err)
		c.JSON(status, payload)
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