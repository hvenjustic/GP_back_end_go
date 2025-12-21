package health

import (
	"net/http"

	"GP_back_end_go/models/dto"

	"github.com/gin-gonic/gin"
)

func CheckHealth(c *gin.Context) {
	c.JSON(http.StatusOK, dto.HealthStatusResponse{
		Status:  "ok",
		Service: "llm-api",
	})
}
