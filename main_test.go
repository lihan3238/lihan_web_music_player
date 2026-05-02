package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIndexUsesOpenToolHubBasePathForAssetsAndRuntimeCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appConfig = config{
		Mode:        "toolbox",
		TemplateDir: filepath.Join(cwd, "templates"),
	}

	router := gin.New()
	router.LoadHTMLGlob(filepath.Join(appConfig.TemplateDir, "*"))
	router.GET("/", index)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-OpenToolHub-Base-Path", "/tools/tool-1/app")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	expectedSnippets := []string{
		`href="/tools/tool-1/app/static/style.css"`,
		`data-base-path="/tools/tool-1/app"`,
		`fetch(appUrl("api/songs"))`,
		`audio.src = appUrl(songs[current].file)`,
		`await loadLyrics(appUrl(songs[current].lrc))`,
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("rendered index missing %q\n%s", snippet, body)
		}
	}
}

func TestToolboxModeRequiresValidOpenToolHubRuntimeJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	appConfig = config{
		Mode:      "toolbox",
		ToolID:    "tool-1",
		Issuer:    "opentoolhub",
		Audience:  "opentoolhub-runtime",
		JWTSecret: "test-secret",
	}

	router := gin.New()
	protected := router.Group("/")
	protected.Use(requirePlatformJWT())
	protected.GET("/api/songs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/songs", nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token to return 401, got %d", missing.Code)
	}

	valid := httptest.NewRecorder()
	validRequest := httptest.NewRequest(http.MethodGet, "/api/songs", nil)
	validRequest.Header.Set("Authorization", "Bearer "+testRuntimeJWT(t, "tool-1", []string{"validate"}))
	router.ServeHTTP(valid, validRequest)
	if valid.Code != http.StatusOK {
		t.Fatalf("expected valid token to return 200, got %d: %s", valid.Code, valid.Body.String())
	}

	wrongTool := httptest.NewRecorder()
	wrongToolRequest := httptest.NewRequest(http.MethodGet, "/api/songs", nil)
	wrongToolRequest.Header.Set("Authorization", "Bearer "+testRuntimeJWT(t, "other-tool", []string{"web_app"}))
	router.ServeHTTP(wrongTool, wrongToolRequest)
	if wrongTool.Code != http.StatusForbidden {
		t.Fatalf("expected wrong tool token to return 403, got %d", wrongTool.Code)
	}

	wrongScope := httptest.NewRecorder()
	wrongScopeRequest := httptest.NewRequest(http.MethodGet, "/api/songs", nil)
	wrongScopeRequest.Header.Set("Authorization", "Bearer "+testRuntimeJWT(t, "tool-1", []string{"execute"}))
	router.ServeHTTP(wrongScope, wrongScopeRequest)
	if wrongScope.Code != http.StatusForbidden {
		t.Fatalf("expected wrong scope token to return 403, got %d", wrongScope.Code)
	}
}

func testRuntimeJWT(t *testing.T, toolID string, scopes []string) string {
	t.Helper()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]any{
		"iss":             appConfig.Issuer,
		"aud":             appConfig.Audience,
		"tool_id":         toolID,
		"tool_version_id": "version-1",
		"tool_slug":       "music-player",
		"scope":           scopes,
		"iat":             int64(1),
		"exp":             int64(4102444800),
		"jti":             "test-token",
	}
	return signTestJWT(t, header, payload)
}

func signTestJWT(t *testing.T, header map[string]string, payload map[string]any) string {
	t.Helper()
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	mac := hmac.New(sha256.New, []byte(appConfig.JWTSecret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
