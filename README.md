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

Create a `.env` file in the project root to configure the port:

```env
PORT=9898
```

The `PORT` value must be between `1` and `65535`. If it is not set, the API uses port `9898`.

```bash
go run main.go
```

Server starts on:

```text
http://localhost:9898
```

### Run in production on Windows

Build a standalone executable instead of using `go run`:

```powershell
go build -trimpath -ldflags="-s -w" -o yt-transcript-api.exe .
```

Keep `.env` beside `yt-transcript-api.exe`, then start it:

```powershell
.\yt-transcript-api.exe
```

For a background process that writes logs to files:

```powershell
Start-Process -FilePath .\yt-transcript-api.exe `
  -RedirectStandardOutput .\server.log `
  -RedirectStandardError .\server-error.log
```

Verify that it is running:

```powershell
Invoke-RestMethod http://localhost:9898/health
```

The application runs Gin in release mode and does not trust forwarded proxy
headers by default. If you expose it to the internet, prefer placing it
behind a reverse proxy such as IIS or Caddy for HTTPS, and open only the
proxy's port in Windows Firewall.

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
- `fulltext` (boolean-like string, optional): return merged transcript text
  - default: `false`
  - use `true` to get merged text output

#### Example: Structured Transcript (default)

```bash
curl "http://localhost:9898/transcript/dQw4w9WgXcQ?lang=en"
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
curl "http://localhost:9898/transcript/dQw4w9WgXcQ?lang=en&fulltext=true"
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

If transcript retrieval fails (for example unavailable subtitles or upstream errors):

```json
{
  "error": "<upstream error message>"
}
```

Status code: `404`

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
Invoke-RestMethod "http://localhost:9898/health"
Invoke-RestMethod "http://localhost:9898/transcript/dQw4w9WgXcQ?lang=en"
Invoke-RestMethod "http://localhost:9898/transcript/dQw4w9WgXcQ?lang=en&fulltext=true"
```

## Notes

- This API expects a plain YouTube video ID in the route (not a full YouTube URL).
- `fulltext=true` concatenates snippet text with spaces.
- Language availability depends on the target video subtitles.
