package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "homework04/internal/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	return c, rec
}

func TestSuccess(t *testing.T) {
	c, rec := setupContext()
	Success(c, map[string]string{"foo": "bar"})

	assert.Equal(t, http.StatusOK, rec.Code)
	var body Body
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
}

func TestFail_AppError_UsesItsOwnStatusAndCode(t *testing.T) {
	c, rec := setupContext()
	Fail(c, apperrors.New(20001, "文章不存在", http.StatusNotFound))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	var body Body
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 20001, body.Code)
	assert.Equal(t, "文章不存在", body.Message)
}

func TestFail_AppError_ThroughWrappedChain(t *testing.T) {
	c, rec := setupContext()
	base := apperrors.New(10001, "用户名已被占用", http.StatusConflict)
	Fail(c, errors.Join(base)) // errors.Join 也保留 errors.As 可达性

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestFail_UnknownError_FallsBackToInternal(t *testing.T) {
	c, rec := setupContext()
	Fail(c, errors.New("something exploded"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var body Body
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, apperrors.CodeInternal, body.Code)
	assert.NotContains(t, rec.Body.String(), "something exploded", "内部错误细节不能泄露给客户端")
}
