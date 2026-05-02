package main

import (
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
