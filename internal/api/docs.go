package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterDocs(r *gin.Engine) {
	r.GET("/docs", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<!doctype html>
<html><head><title>Evently Docs</title></head>
<body>
<redoc spec-url="/openapi.yaml"></redoc>
<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body></html>`)
	})
	r.StaticFile("/openapi.yaml", "docs/openapi.yaml")
}
