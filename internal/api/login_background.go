package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const maxLoginBackgroundSize = 10 << 20

var allowedLoginBackgroundTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

func (s *Server) loginBackgroundPath() string {
	return filepath.Join(s.store.GetDataDir(), "login-background")
}

func (s *Server) getLoginBackground(c *gin.Context) {
	data, err := os.ReadFile(s.loginBackgroundPath())
	if errors.Is(err, os.ErrNotExist) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取登录背景失败"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, http.DetectContentType(data), data)
}

func (s *Server) uploadLoginBackground(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片"})
		return
	}
	if fileHeader.Size > maxLoginBackgroundSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片不能超过 10MB"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取图片"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxLoginBackgroundSize+1))
	if err != nil || len(data) > maxLoginBackgroundSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片不能超过 10MB"})
		return
	}
	if len(data) == 0 || !allowedLoginBackgroundTypes[http.DetectContentType(data)] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG、PNG、WEBP"})
		return
	}
	temporary, err := os.CreateTemp(s.store.GetDataDir(), "login-background-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存登录背景失败"})
		return
	}
	_, err = temporary.Write(data)
	if err == nil {
		err = temporary.Chmod(0644)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary.Name(), s.loginBackgroundPath())
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存登录背景失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "登录背景已更新"})
}

func (s *Server) deleteLoginBackground(c *gin.Context) {
	err := os.Remove(s.loginBackgroundPath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复默认背景失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已恢复默认背景"})
}
