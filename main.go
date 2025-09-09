package main

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html/charset"
)

func main() {
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	// 首页
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 获取音乐列表
	r.GET("/api/songs", func(c *gin.Context) {
		files, _ := ioutil.ReadDir("./music")
		var songs []map[string]string

		for _, f := range files {
			if !f.IsDir() && isAudioFile(f.Name()) {
				base := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
				songs = append(songs, map[string]string{
					"title": base,
					"file":  "/music/" + f.Name(),
					"lrc":   "/api/lyrics/" + base,
				})
			}
		}
		c.JSON(200, songs)
	})

	// 获取歌词（自动转 UTF-8）
	r.GET("/api/lyrics/:name", func(c *gin.Context) {
		name := c.Param("name")
		var content []byte
		var err error

		// 支持多种扩展名
		for _, ext := range []string{".lrc", ".txt"} {
			path := filepath.Join("music", name+ext)
			content, err = ioutil.ReadFile(path)
			if err == nil {
				break
			}
		}
		if err != nil {
			c.String(404, "")
			return
		}

		// 自动转 UTF-8
		utf8Reader, err := charset.NewReaderLabel("utf-8", strings.NewReader(string(content)))
		if err == nil {
			data, _ := ioutil.ReadAll(utf8Reader)
			c.String(200, string(data))
			return
		}
		c.String(200, string(content))
	})

	// ✅ 音乐文件接口：支持缓存 + 断点续传
	r.GET("/music/:file", func(c *gin.Context) {
		file := c.Param("file")
		path := filepath.Join("music", file)

		// 设置缓存头（浏览器下次直接用缓存）
		c.Header("Cache-Control", "public, max-age=86400")

		// 交给 net/http 处理，支持 Range 请求（断点续传）
		http.ServeFile(c.Writer, c.Request, path)
	})

	r.Run(":8080")
}

func isAudioFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".mp3", ".m4a", ".mp4", ".ogg", ".wav", ".flac":
		return true
	}
	return false
}
