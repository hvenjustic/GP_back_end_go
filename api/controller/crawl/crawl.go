package crawl

import (
	"errors"
	"net/http"
	"strconv"

	"GP_back_end_go/internal/service"
	"GP_back_end_go/models/dto"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SubmitTasks(c *gin.Context) {
	var req dto.SubmitTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error(), Code: http.StatusBadRequest})
		return
	}

	resp, err := service.SubmitCrawlTasks(c.Request.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrBadRequest):
			status = http.StatusBadRequest
		case errors.Is(err, service.ErrRedisUnavailable):
			status = http.StatusInternalServerError
		}
		c.JSON(status, dto.ErrorResponse{Error: err.Error(), Code: status})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func GetStatus(c *gin.Context) {
	resp, err := service.GetCrawlStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error(), Code: http.StatusInternalServerError})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func ClearQueue(c *gin.Context) {
	var req dto.ClearQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error(), Code: http.StatusBadRequest})
		return
	}
	resp, err := service.ClearQueue(c.Request.Context(), req.QueueName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrBadRequest) {
			status = http.StatusBadRequest
		}
		c.JSON(status, dto.ErrorResponse{Error: err.Error(), Code: status})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func PostTaskResult(c *gin.Context) {
	var req dto.TaskResultCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error(), Code: http.StatusBadRequest})
		return
	}

	resp, err := service.ApplyTaskResult(c.Request.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrBadRequest):
			status = http.StatusBadRequest
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = http.StatusNotFound
		}
		c.JSON(status, dto.ErrorResponse{Error: err.Error(), Code: status})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func ListResults(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	resp, err := service.ListCrawlResults(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error(), Code: http.StatusInternalServerError})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func GetResultDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "id invalid", Code: http.StatusBadRequest})
		return
	}

	resp, err := service.GetResultDetail(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, service.ErrBadRequest):
			status = http.StatusBadRequest
		case errors.Is(err, gorm.ErrRecordNotFound):
			status = http.StatusNotFound
		}
		c.JSON(status, dto.ErrorResponse{Error: err.Error(), Code: status})
		return
	}

	c.JSON(http.StatusOK, resp)
}
