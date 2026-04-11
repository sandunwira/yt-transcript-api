# YouTube Transcription API (Go)

A lightweight Go API for fetching YouTube transcripts.

Built with:

- [Gin](https://github.com/gin-gonic/gin) for HTTP routing
- [youtube_transcript_api (Go)](https://github.com/dogslee/youtube_transcript_api) for transcript retrieval

## Features

- Fetch transcript snippets for a YouTube video
- Return transcript as either:
  - structured snippets (`fulltext=false`, default)
  - a single combined text body (`fulltext=true`)
- Select transcript language via query parameter
- Health check endpoint

## Requirements

- Go 1.26.1 (or compatible with `go.mod`)

## Project Structure

- `main.go` - API server and handlers
- `go.mod` - module and dependencies

## Install Dependencies

```bash
go mod download
```

## Run the API

```bash
go run main.go
```

Server starts on:

```text
http://localhost:8080
```

## API Endpoints

### 1) Health Check

**GET** `/health`

Response:

```json
{
  "status": "ok"
}
```

### 2) Get Transcript

**GET** `/transcript/:id`

#### Path Parameter

- `id` (string, required): YouTube video ID (must be exactly 11 characters)

#### Query Parameters

- `lang` (string, optional): transcript language code
  - default: `en`
  - supports comma-separated fallback order, for example: `lang=en,es,de`
- `fulltext` (boolean-like string, optional): return merged transcript text
  - default: `false`
  - use `true` to get merged text output

#### Example: Structured Transcript (default)

```bash
curl "http://localhost:8080/transcript/dQw4w9WgXcQ?lang=en"
```

Example response:

```json
{
  "video_id": "dQw4w9WgXcQ",
  "language": "en",
  "is_generated": false,
  "snippets": [
    {
      "Text": "We're no strangers to love",
      "Start": 12.34,
      "Duration": 2.56
    }
  ]
}
```

#### Example: Full Text Transcript

```bash
curl "http://localhost:8080/transcript/dQw4w9WgXcQ?lang=en&fulltext=true"
```

Example response:

```json
{
  "video_id": "dQw4w9WgXcQ",
  "title": "Never Gonna Give You Up",
  "fulltext": "We're no strangers to love You know the rules and so do I ..."
}
```

## Error Responses

### Invalid Video ID

If `id` is not 11 characters:

```json
{
  "error": "Invalid YouTube video ID"
}
```

Status code: `400`

### Transcript Not Found / Fetch Error

If transcript retrieval fails because subtitles do not exist for requested language(s):

```json
{
  "error": "No transcript found for the requested language(s)",
  "cause": "<short explanation>"
}
```

Status code: `404`

### Age Restricted Video

If the target video is age-restricted:

```json
{
  "error": "This video is age-restricted",
  "cause": "<short explanation>"
}
```

Status code: `403`

### Request Blocked by YouTube (Cloud IP)

If YouTube blocks the server IP (common on cloud platforms):

```json
{
  "error": "YouTube blocked requests from this server IP",
  "hint": "Configure YT_PROXY_HTTP and/or YT_PROXY_HTTPS with a residential rotating proxy, then retry",
  "cause": "<short explanation>"
}
```

Status code: `429`

### Upstream YouTube Failure

If YouTube requests fail for non-404 reasons:

```json
{
  "error": "Failed to fetch transcript from YouTube",
  "cause": "<short explanation>"
}
```

Status code: `502`

### Internal Server Error

If API client creation fails:

```json
{
  "error": "Failed to create API client"
}
```

Status code: `500`

## Quick Test (PowerShell)

```powershell
Invoke-RestMethod "http://localhost:8080/health"
Invoke-RestMethod "http://localhost:8080/transcript/dQw4w9WgXcQ?lang=en"
Invoke-RestMethod "http://localhost:8080/transcript/dQw4w9WgXcQ?lang=en&fulltext=true"
```

## Notes

- This API expects a plain YouTube video ID in the route (not a full YouTube URL).
- `fulltext=true` concatenates snippet text with spaces.
- Language availability depends on the target video subtitles.

## Proxy Configuration (Optional)

To reduce YouTube IP blocking on cloud hosts (for example Render), set one or both environment variables:

- `YT_PROXY_HTTP`
- `YT_PROXY_HTTPS`

Example:

```bash
YT_PROXY_HTTP=http://user:pass@proxy-host:port
YT_PROXY_HTTPS=http://user:pass@proxy-host:port
```
