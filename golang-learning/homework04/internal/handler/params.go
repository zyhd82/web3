package handler

import (
	"strconv"

	apperrors "homework04/internal/pkg/errors"
	"homework04/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// parseIDParam 从路径参数解析出 uint 主键，失败时直接写统一错误响应，调用方需立刻 return。
func parseIDParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		response.Fail(c, apperrors.ErrInvalidParams)
		return 0, false
	}
	return uint(id), true
}

// parsePagination 从 query 解析分页参数；缺省/非法值由 repository 层的 normalizePage 兜底纠正
// （见 docs/DATABASE_CONVENTIONS.md 分页一节），这里只负责把字符串转成 int。
func parsePagination(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	return page, pageSize
}
