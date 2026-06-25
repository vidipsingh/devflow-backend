package api

import (
    "devflow-backend/internal/api/response"
    "github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data interface{})      { response.OK(c, data) }
func Created(c *gin.Context, data interface{}) { response.Created(c, data) }
func BadRequest(c *gin.Context, msg string)    { response.BadRequest(c, msg) }
func NotFound(c *gin.Context, msg string)      { response.NotFound(c, msg) }
func InternalError(c *gin.Context, msg string) { response.InternalError(c, msg) }
func Unauthorized(c *gin.Context)              { response.Unauthorized(c) }
