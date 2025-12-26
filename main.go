package main

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html/charset"
)

var musicDirs = []string{
	"./music",
	"./musics",
}

func main() {
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 🎵 获取音乐列表（music + musics 并集）
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

		var songs []map[string]string
		for _, v := range songMap {
			songs = append(songs, v)
		}

		c.JSON(200, songs)
	})

	// 📄 获取歌词（music / musics 任意一个）
	r.GET("/api/lyrics/:name", func(c *gin.Context) {
		name := c.Param("name")
		var content []byte
		var err error

		for _, dir := range musicDirs {
			for _, ext := range []string{".lrc", ".txt"} {
				path := filepath.Join(dir, name+ext)
				content, err = ioutil.ReadFile(path)
				if err == nil {
					goto FOUND
				}
			}
		}

		c.String(404, "")
		return

	FOUND:
		utf8Reader, err := charset.NewReaderLabel("utf-8", strings.NewReader(string(content)))
		if err == nil {
			data, _ := ioutil.ReadAll(utf8Reader)
			c.String(200, string(data))
			return
		}
		c.String(200, string(content))
	})

	// 🎧 音乐文件接口（支持 music / musics）
	r.GET("/music/:file", func(c *gin.Context) {
		file := c.Param("file")

		for _, dir := range musicDirs {
			path := filepath.Join(dir, file)
			if _, err := ioutil.ReadFile(path); err == nil {
				c.Header("Cache-Control", "public, max-age=86400")
				http.ServeFile(c.Writer, c.Request, path)
				return
			}
		}

		c.Status(404)
	})

	r.Run(":8080")
}

func isAudioFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".mp4", ".ogg", ".wav", ".flac":
		return true
	}
	return false
}
