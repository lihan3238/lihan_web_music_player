package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html/charset"
)

type config struct {
	Mode        string
	Port        string
	ToolID      string
	Issuer      string
	Audience    string
	JWTSecret   string
	MusicDirs   []string
	StaticDir   string
	TemplateDir string
}

type jwtClaims struct {
	Issuer    string   `json:"iss"`
	Audience  string   `json:"aud"`
	ToolID    string   `json:"tool_id"`
	ToolSlug  string   `json:"tool_slug"`
	Scope     []string `json:"scope"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	ID        string   `json:"jti"`
}

var appConfig config

func main() {
	appConfig = loadConfig()

	r := gin.Default()
	r.LoadHTMLGlob(filepath.Join(appConfig.TemplateDir, "*"))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "mode": appConfig.Mode})
	})
	r.GET("/meta", func(c *gin.Context) {
		c.JSON(http.StatusOK, protocolMeta())
	})

	protected := r.Group("/")
	if appConfig.Mode == "toolbox" {
		protected.Use(requirePlatformJWT())
	}
	protected.Static("/static", appConfig.StaticDir)
	protected.GET("/", index)
	protected.GET("/api/songs", songs)
	protected.GET("/api/lyrics/:name", lyrics)
	protected.GET("/music/:file", music)

	if err := r.Run(":" + appConfig.Port); err != nil {
		panic(err)
	}
}

func loadConfig() config {
	resourceRoot := findResourceRoot()
	port := env("PORT", "8080")
	flag.StringVar(&port, "port", port, "server port")
	flag.Parse()

	musicDirs := []string{}
	if musicDir := os.Getenv("MUSIC_DIR"); strings.TrimSpace(musicDir) != "" {
		for _, dir := range strings.Split(musicDir, string(os.PathListSeparator)) {
			if strings.TrimSpace(dir) != "" {
				musicDirs = append(musicDirs, dir)
			}
		}
	}
	musicDirs = append(musicDirs, filepath.Join(resourceRoot, "music"), filepath.Join(resourceRoot, "musics"))

	return config{
		Mode:        env("OPENTOOLHUB_MODE", "standalone"),
		Port:        port,
		ToolID:      env("OPENTOOLHUB_TOOL_ID", "lihan-web-music-player"),
		Issuer:      env("OPENTOOLHUB_ISSUER", "opentoolhub"),
		Audience:    env("OPENTOOLHUB_AUDIENCE", "opentoolhub-runtime"),
		JWTSecret:   env("OPENTOOLHUB_JWT_SECRET", env("RUNTIME_JWT_SECRET", "opentoolhub-dev-runtime-secret-change-me")),
		MusicDirs:   musicDirs,
		StaticDir:   filepath.Join(resourceRoot, "static"),
		TemplateDir: filepath.Join(resourceRoot, "templates"),
	}
}

func findResourceRoot() string {
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "templates", "index.html")); err == nil {
			return cwd
		}
	}
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(exePath)
}

func protocolMeta() gin.H {
	return gin.H{
		"protocol_version": "opentoolhub.tool.v3",
		"tool_type":        "WEB_APP",
		"name":             "Lihan Web Music Player",
		"version":          "1.0.0",
		"description":      "A real local music player that can run standalone or behind the OpenToolHub WEB_APP proxy.",
		"source": gin.H{
			"repository_url": "https://github.com/lihan3238/lihan_web_music_player",
			"license":        "MIT",
			"deploy":         "OPENTOOLHUB_MODE=toolbox MUSIC_DIR=/mnt/c/lihan_work/github_repos/music/musics OPENTOOLHUB_TOOL_ID=<tool-id> RUNTIME_JWT_SECRET=<runtime-secret> go run .",
			"directory":      ".",
		},
		"auth": gin.H{
			"type":     "platform_jwt",
			"issuer":   appConfig.Issuer,
			"audience": appConfig.Audience,
		},
		"execution": gin.H{
			"mode":                "sync",
			"input_content_type":  "application/json",
			"output_content_type": "application/json",
		},
		"web_entry_url": "/",
		"input_schema":  gin.H{"type": "object", "properties": gin.H{}},
		"output_schema": gin.H{"type": "object", "properties": gin.H{}},
		"validation": gin.H{
			"sample_input":    gin.H{},
			"web_probe_paths": []string{"/", "/api/songs"},
		},
	}
}

func index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"BasePath": strings.TrimRight(c.GetHeader("X-OpenToolHub-Base-Path"), "/"),
		"Mode":     appConfig.Mode,
	})
}

func songs(c *gin.Context) {
	songMap := make(map[string]map[string]string)
	for _, dir := range appConfig.MusicDirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() || !isAudioFile(file.Name()) {
				continue
			}
			base := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			if _, exists := songMap[base]; exists {
				continue
			}
			escapedFile := strings.ReplaceAll(file.Name(), "\\", "/")
			songMap[base] = map[string]string{
				"title": base,
				"file":  "music/" + escapedFile,
				"lrc":   "api/lyrics/" + base,
			}
		}
	}

	list := make([]map[string]string, 0, len(songMap))
	for _, song := range songMap {
		list = append(list, song)
	}
	c.JSON(http.StatusOK, list)
}

func lyrics(c *gin.Context) {
	content, err := findLyricFile(c.Param("name"))
	if err != nil {
		c.String(http.StatusNotFound, "")
		return
	}
	reader, err := charset.NewReaderLabel("utf-8", strings.NewReader(string(content)))
	if err == nil {
		data, _ := io.ReadAll(reader)
		c.String(http.StatusOK, string(data))
		return
	}
	c.String(http.StatusOK, string(content))
}

func music(c *gin.Context) {
	file := filepath.Base(c.Param("file"))
	for _, dir := range appConfig.MusicDirs {
		path := filepath.Join(dir, file)
		if _, err := os.Stat(path); err == nil {
			c.Header("Cache-Control", "public, max-age=86400")
			http.ServeFile(c.Writer, c.Request, path)
			return
		}
	}
	c.Status(http.StatusNotFound)
}

func requirePlatformJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" || token == c.GetHeader("Authorization") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing OpenToolHub runtime token"})
			return
		}
		claims, err := verifyJWT(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": err.Error()})
			return
		}
		if claims.Issuer != appConfig.Issuer || claims.Audience != appConfig.Audience {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "runtime token issuer or audience mismatch"})
			return
		}
		if claims.ToolID != appConfig.ToolID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "runtime token tool mismatch"})
			return
		}
		if !hasAnyScope(claims.Scope, "validate", "web_app") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "runtime token scope mismatch"})
			return
		}
		if claims.ExpiresAt < time.Now().Unix() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "runtime token expired"})
			return
		}
		c.Next()
	}
}

func verifyJWT(token string) (jwtClaims, error) {
	var claims jwtClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, errString("invalid runtime token")
	}
	expected := sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return claims, errString("runtime token signature mismatch")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, errString("invalid runtime token payload")
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, errString("invalid runtime token claims")
	}
	return claims, nil
}

func sign(value string) string {
	mac := hmac.New(sha256.New, []byte(appConfig.JWTSecret))
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func hasAnyScope(scopes []string, allowed ...string) bool {
	for _, scope := range scopes {
		for _, candidate := range allowed {
			if scope == candidate {
				return true
			}
		}
	}
	return false
}

type errString string

func (e errString) Error() string { return string(e) }

func env(key string, fallback string) string {
	if value := os.Getenv(key); strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func isAudioFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".mp4", ".ogg", ".wav", ".flac":
		return true
	default:
		return false
	}
}

func findLyricFile(name string) ([]byte, error) {
	for _, dir := range appConfig.MusicDirs {
		for _, ext := range []string{".lrc", ".txt"} {
			path := filepath.Join(dir, name+ext)
			if _, err := os.Stat(path); err == nil {
				return os.ReadFile(path)
			}
		}
	}
	return nil, os.ErrNotExist
}
