package server

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed assets/console.html
var panelAssets embed.FS

func (s *Server) consoleHandler(c *gin.Context) {
	data, err := panelAssets.ReadFile("assets/console.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load backup console")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
