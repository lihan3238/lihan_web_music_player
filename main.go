package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html/charset"
)

// 🎵 多音乐目录（并集）——基于 exe 目录
var musicDirs []string

func main() {
	// ========= 获取 exe 所在目录 =========
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exePath)

	// ========= 初始化资源路径 =========
	musicDirs = []string{
		filepath.Join(exeDir, "music"),
		filepath.Join(exeDir, "musics"),
	}

	staticDir := filepath.Join(exeDir, "static")
	templateDir := filepath.Join(exeDir, "templates", "*")

	// ========= 端口配置（参数 > 环境变量 > 默认） =========
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	flag.StringVar(&port, "port", port, "server port")
	flag.Parse()

	// ========= Gin 初始化 =========
	r := gin.Default()
	r.Static("/static", staticDir)
	r.LoadHTMLGlob(templateDir)

	// ========= 首页 =========
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// ========= 获取音乐列表（并集 + 去重） =========
	r.GET("/api/songs", func(c *gin.Context) {
		songMap := make(map[string]map[string]string)

		for _, dir := range musicDirs {
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				continue
			}

			for _, f := range files {
				if f.IsDir() || !isAudioFile(f.Name()) {
					continue
				}

				base := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
				if _, exists := songMap[base]; exists {
					continue
				}

				songMap[base] = map[string]string{
					"title": base,
					"file":  "/music/" + f.Name(),
					"lrc":   "/api/lyrics/" + base,
				}
			}
		}

		songs := make([]map[string]string, 0, len(songMap))
		for _, v := range songMap {
			songs = append(songs, v)
		}

		c.JSON(http.StatusOK, songs)
	})

	// ========= 获取歌词 =========
	r.GET("/api/lyrics/:name", func(c *gin.Context) {
		name := c.Param("name")

		content, err := findLyricFile(name)
		if err != nil {
			c.String(http.StatusNotFound, "")
			return
		}

		utf8Reader, err := charset.NewReaderLabel("utf-8", strings.NewReader(string(content)))
		if err == nil {
			data, _ := ioutil.ReadAll(utf8Reader)
			c.String(http.StatusOK, string(data))
			return
		}

		c.String(http.StatusOK, string(content))
	})

	// ========= 音乐文件服务 =========
	r.GET("/music/:file", func(c *gin.Context) {
		file := c.Param("file")

		for _, dir := range musicDirs {
			path := filepath.Join(dir, file)
			if _, err := os.Stat(path); err == nil {
				c.Header("Cache-Control", "public, max-age=86400")
				http.ServeFile(c.Writer, c.Request, path)
				return
			}
		}

		c.Status(http.StatusNotFound)
	})

	// ========= 启动 =========
	r.Run(":" + port)
}

// ================= 工具函数 =================

func isAudioFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".mp4", ".ogg", ".wav", ".flac":
		return true
	}
	return false
}

func findLyricFile(name string) ([]byte, error) {
	for _, dir := range musicDirs {
		for _, ext := range []string{".lrc", ".txt"} {
			path := filepath.Join(dir, name+ext)
			if _, err := os.Stat(path); err == nil {
				return ioutil.ReadFile(path)
			}
		}
	}
	return nil, os.ErrNotExist
}
