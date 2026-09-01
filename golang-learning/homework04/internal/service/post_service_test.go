package service

import (
	"context"
	"errors"
	"testing"

	apperrors "homework04/internal/pkg/errors"

	"homework04/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Delete 的成功路径需要真实的 *gorm.DB 事务（PostService.Delete 直接用 repository 包构造事务范围
// 内的实例，见 post_service.go 的注释），mock 测不了，由 Step 6/7 写的手工冒烟测试覆盖过。
// 这里只测 Delete 在真正开事务之前就能短路返回的两个分支（not found / forbidden）。

func TestPostService_Create_Success(t *testing.T) {
	repo := new(mockPostRepo)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).
		Run(func(args mock.Arguments) {
			p := args.Get(1).(*model.Post)
			p.ID = 1
		}).Return(nil)
	repo.On("FindByID", mock.Anything, uint(1)).
		Return(&model.Post{ID: 1, UserID: 10, Title: "标题", Content: "正文"}, nil)

	svc := NewPostService(repo, nil)
	post, err := svc.Create(context.Background(), 10, "标题", "正文")

	require.NoError(t, err)
	assert.Equal(t, uint(1), post.ID)
}

func TestPostService_Create_RepoFails(t *testing.T) {
	repo := new(mockPostRepo)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Post")).Return(errors.New("db down"))

	svc := NewPostService(repo, nil)
	_, err := svc.Create(context.Background(), 10, "标题", "正文")

	require.Error(t, err)
}

func TestPostService_Update_ByOwner(t *testing.T) {
	post := &model.Post{ID: 1, UserID: 10, Title: "旧标题", Content: "旧正文"}
	repo := new(mockPostRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(post, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Post")).Return(nil)

	newTitle := "新标题"
	svc := NewPostService(repo, nil)
	got, err := svc.Update(context.Background(), 1, 10, model.RoleUser, &newTitle, nil)

	require.NoError(t, err)
	assert.Equal(t, "新标题", got.Title)
	assert.Equal(t, "旧正文", got.Content, "没传的字段不应该被清空")
}

func TestPostService_Update_ByAdmin(t *testing.T) {
	post := &model.Post{ID: 1, UserID: 10, Title: "旧标题"}
	repo := new(mockPostRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(post, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Post")).Return(nil)

	newTitle := "管理员改的标题"
	svc := NewPostService(repo, nil)
	got, err := svc.Update(context.Background(), 1, 999, model.RoleAdmin, &newTitle, nil)

	require.NoError(t, err)
	assert.Equal(t, "管理员改的标题", got.Title)
}

func TestPostService_Update_Forbidden(t *testing.T) {
	post := &model.Post{ID: 1, UserID: 10, Title: "旧标题"}
	repo := new(mockPostRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(post, nil)

	newTitle := "想改别人的标题"
	svc := NewPostService(repo, nil)
	_, err := svc.Update(context.Background(), 1, 999, model.RoleUser, &newTitle, nil)

	assert.ErrorIs(t, err, apperrors.ErrForbidden)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPostService_Update_NotFound(t *testing.T) {
	repo := new(mockPostRepo)
	repo.On("FindByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	newTitle := "x"
	svc := NewPostService(repo, nil)
	_, err := svc.Update(context.Background(), 999, 10, model.RoleUser, &newTitle, nil)

	assert.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostService_Delete_NotFound(t *testing.T) {
	repo := new(mockPostRepo)
	repo.On("FindByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	svc := NewPostService(repo, nil) // db=nil：这条路径不会走到事务
	err := svc.Delete(context.Background(), 999, 10, model.RoleUser)

	assert.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostService_Delete_Forbidden(t *testing.T) {
	post := &model.Post{ID: 1, UserID: 10}
	repo := new(mockPostRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(post, nil)

	svc := NewPostService(repo, nil) // db=nil：权限检查会在开事务前短路返回
	err := svc.Delete(context.Background(), 1, 999, model.RoleUser)

	assert.ErrorIs(t, err, apperrors.ErrForbidden)
}

func TestPostService_FindByID(t *testing.T) {
	repo := new(mockPostRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(&model.Post{ID: 1, Title: "标题"}, nil)

	svc := NewPostService(repo, nil)
	post, err := svc.FindByID(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, "标题", post.Title)
}

func TestPostService_List(t *testing.T) {
	repo := new(mockPostRepo)
	posts := []*model.Post{{ID: 1}, {ID: 2}}
	repo.On("List", mock.Anything, 1, 10, (*uint)(nil)).Return(posts, int64(2), nil)

	svc := NewPostService(repo, nil)
	got, total, err := svc.List(context.Background(), 1, 10, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, got, 2)
}
