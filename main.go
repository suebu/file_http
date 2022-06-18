package main

import (
	"github.com/gin-gonic/gin"
	"go_demo/router"
)

func main() {
	r := gin.Default()

	r = router.HttpRouter(r)
	panic(r.Run())
}
