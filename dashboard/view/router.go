package view

import (
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RouterList(c *gin.Context) {
	var resp BaseResponse
	var request struct {
		Offset int `json:"offset"`
		Limit int `json:"limit"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Exception(err)
		resp.InitError(errors.New(8))
		c.JSON(http.StatusBadRequest, resp)
		return
	}


}
