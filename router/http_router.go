package router

import (
	"github.com/gin-gonic/gin"
	"go_demo/controller"
)

func HttpRouter(r *gin.Engine) *gin.Engine {
	r.POST("/api/download", controller.Download)
	return r
}
