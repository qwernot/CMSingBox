package api

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxBackupSize = 10 << 20

func (s *Server) exportBackup(c *gin.Context) {
	data, err := s.store.ExportConfiguration()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: "data.json", Method: zip.Deflate}
	header.SetModTime(time.Now())
	file, err := archive.CreateHeader(header)
	if err == nil {
		_, err = file.Write(data)
	}
	if closeErr := archive.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建备份失败"})
		return
	}
	filename := fmt.Sprintf("cmsingbox-backup-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/zip", buffer.Bytes())
}

func readBackupJSON(file multipart.File) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(file, maxBackupSize+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBackupSize {
		return nil, fmt.Errorf("备份文件不能超过 10MB")
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("备份文件不是有效的 ZIP")
	}
	for _, entry := range archive.File {
		if strings.TrimPrefix(entry.Name, "./") != "data.json" || entry.FileInfo().IsDir() {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, maxBackupSize+1))
		reader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if len(data) > maxBackupSize {
			return nil, fmt.Errorf("备份数据不能超过 10MB")
		}
		return data, nil
	}
	return nil, fmt.Errorf("备份中缺少 data.json")
}

func (s *Server) importBackup(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择备份文件"})
		return
	}
	if fileHeader.Size > maxBackupSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备份文件不能超过 10MB"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取备份文件"})
		return
	}
	defer file.Close()
	data, err := readBackupJSON(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.license != nil {
		s.store.SetSubscriptionLimit(s.license.Limit())
	}
	if err := s.store.RestoreConfiguration(data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "备份已恢复，请重新应用配置"})
}
