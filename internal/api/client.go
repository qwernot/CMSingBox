package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) clientConfig(c *gin.Context) {
	expected, actual := s.store.GetSettings().ClientConfigPath, c.Param("path")
	if len(expected) != len(actual) || subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) != 1 {
		c.Status(http.StatusNotFound)
		return
	}
	config, err := s.buildConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成客户端配置失败"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=sing-box.json")
	c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(config))
}
