package respond

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/apierror"
)

type successEnvelope struct {
	Code int `json:"code"`
	Data any `json:"data"`
}

type errorEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func Success(c *gin.Context, status int, data any) {
	c.Set("business_code", 0)
	c.JSON(status, successEnvelope{Code: 0, Data: data})
}

func OK(c *gin.Context, data any) {
	Success(c, http.StatusOK, data)
}

func Created(c *gin.Context, data any) {
	Success(c, http.StatusCreated, data)
}

func NoData(c *gin.Context) {
	OK(c, nil)
}

func Error(c *gin.Context, err error) {
	appErr := apierror.As(err)
	if appErr.HTTPStatus == 0 {
		appErr.HTTPStatus = http.StatusInternalServerError
	}
	c.Set("business_code", appErr.Code)
	c.AbortWithStatusJSON(appErr.HTTPStatus, errorEnvelope{
		Code: appErr.Code, Message: appErr.Message, Details: appErr.Details,
	})
}
