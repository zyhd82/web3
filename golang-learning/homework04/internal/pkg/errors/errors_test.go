package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_DefaultHTTPStatus(t *testing.T) {
	err := New(10001, "用户不存在")
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus, "不传 httpStatus 时应缺省为 400")
	assert.Equal(t, 10001, err.Code)
	assert.Equal(t, "用户不存在", err.Message)
}

func TestNew_ExplicitHTTPStatus(t *testing.T) {
	err := New(10001, "用户不存在", http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, err.HTTPStatus)
}

func TestAppError_Error(t *testing.T) {
	plain := New(40000, "参数不合法")
	assert.Equal(t, "参数不合法", plain.Error())

	wrapped := Wrap(errors.New("db timeout"), 40003, "服务器内部错误")
	assert.Equal(t, "服务器内部错误: db timeout", wrapped.Error())
}

func TestWrap_UnwrapAndErrorsIs(t *testing.T) {
	sentinel := errors.New("record not found")
	wrapped := Wrap(sentinel, 10004, "用户不存在", http.StatusNotFound)

	assert.ErrorIs(t, wrapped, sentinel, "Wrap 应该保留错误链，errors.Is 能找到底层 sentinel")
	assert.Equal(t, sentinel, errors.Unwrap(wrapped))
}

func TestErrors_As_ThroughFmtErrorfChain(t *testing.T) {
	// 模拟 service 层常见写法：fmt.Errorf("...: %w", appErr)
	base := New(20001, "文章不存在", http.StatusNotFound)
	chained := fmt.Errorf("find post by id: %w", base)

	var got *AppError
	require := assert.New(t)
	require.True(errors.As(chained, &got))
	require.Equal(20001, got.Code)
}

func TestPredefinedErrors_HaveDistinctCodes(t *testing.T) {
	codes := map[int]bool{
		CodeInvalidParams: true,
		CodeUnauthorized:  true,
		CodeForbidden:     true,
		CodeInternal:      true,
	}
	assert.Len(t, codes, 4, "预定义错误码不应该重复")
}
