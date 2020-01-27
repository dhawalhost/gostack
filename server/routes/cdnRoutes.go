package routes

import (
	"github.com/DhawalDN/gostack/server/handlers"
	"github.com/gin-gonic/gin"
)

// InitCdnRoutes :
func InitCdnRoutes(o *gin.RouterGroup) {
	o.POST("/upload", handlers.UploadFileHandler())
	// o.StaticFS("/files/", gin.Dir(handlers.UploadPath, true))
	// o.Static("/files/", handlers.UploadPath)
	// fs := http.FileServer(http.Dir(handlers.UploadPath))
	// http.Handle("/files/", http.StripPrefix("/files", fs))
	// o.POST("/login/registerteam", handlers.RegisterTeam())

}