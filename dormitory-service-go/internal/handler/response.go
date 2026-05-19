package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageData wraps paginated results.
type PageData struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// Success sends a 200 response with payload.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

// Created sends a 201 response with payload.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Code:    http.StatusCreated,
		Message: "created",
		Data:    data,
	})
}

// Error sends an error response with the given HTTP status code and message.
func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, APIResponse{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// PageResult sends a paginated response.
func PageResult(c *gin.Context, items interface{}, total int64, page, size int) {
	Success(c, PageData{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	})
}
