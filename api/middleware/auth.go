package middleware

import (
	"net/http"
	"strings"

	"image-api/internal/mysql"
	"image-api/pkg/log"

	"github.com/gin-gonic/gin"
)

var skipAuthPaths = []string{
	"/check/status",
	"/monitor/pprof",
}

// shouldSkipAuth 检查当前路径是否需要跳过鉴权
func shouldSkipAuth(path string) bool {
	for _, skipPath := range skipAuthPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// isValidTokenFormat 验证token格式是否符合要求
// token必须以"cky_"开头，总长度为24个字符
func isValidTokenFormat(token string) bool {
	// 检查长度
	if len(token) != 24 {
		return false
	}

	// 检查是否以"cky_"开头
	if !strings.HasPrefix(token, "cky_") {
		return false
	}

	return true
}

// AuthMiddleware API Key验证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查当前路径是否需要跳过鉴权
		if shouldSkipAuth(c.Request.URL.Path) {
			log.Info("AuthMiddleware", "skipping auth for path", "path", c.Request.URL.Path)
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Error("AuthMiddleware", "missing authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// 检查Bearer token格式
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Error("AuthMiddleware", "invalid authorization format")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format",
			})
			c.Abort()
			return
		}

		// 提取token
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 字符串级别的预检查：验证token格式
		if !isValidTokenFormat(token) {
			log.Error("AuthMiddleware", "invalid token format", "token", token)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key format",
			})
			c.Abort()
			return
		}

		// 使用数据库验证API Key
		userApiKeyDAO := mysql.NewUserApiKeyDAO()
		isValid, err := userApiKeyDAO.IsApiKeyValid(token)
		if err != nil {
			log.Error("AuthMiddleware", "database error when validating api key", "error", err.Error(), "token", token)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			c.Abort()
			return
		}

		if !isValid {
			log.Error("AuthMiddleware", "invalid or expired api key", "token", token)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired API key",
			})
			c.Abort()
			return
		}

		// 将token存储到上下文中，供后续使用
		c.Set("api_key", token)
		c.Next()
	}
}
