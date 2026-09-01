package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s *Server) previewCleanup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": s.cleaner.Preview()})
}
func (s *Server) runCleanup(c *gin.Context) {
	var req struct {
		Logs      bool `json:"logs"`
		Temporary bool `json:"temporary"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效清理请求"})
		return
	}
	if !req.Logs && !req.Temporary {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择清理项目"})
		return
	}
	if err := s.cleaner.Clean(req.Logs, req.Temporary); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "清理完成", "data": s.cleaner.Preview()})
}
