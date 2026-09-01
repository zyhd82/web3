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

func TestCommentService_Create_Success(t *testing.T) {
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, uint(1)).Return(&model.Post{ID: 1}, nil)

	commentRepo := new(mockCommentRepo)
	commentRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Comment")).
		Run(func(args mock.Arguments) {
			c := args.Get(1).(*model.Comment)
			c.ID = 100
		}).Return(nil)
	commentRepo.On("FindByID", mock.Anything, uint(100)).
		Return(&model.Comment{ID: 100, PostID: 1, UserID: 10, Content: "第一条评论"}, nil)

	svc := NewCommentService(commentRepo, postRepo)
	comment, err := svc.Create(context.Background(), 1, 10, "第一条评论")

	require.NoError(t, err)
	assert.Equal(t, uint(100), comment.ID)
}

func TestCommentService_Create_PostNotFound(t *testing.T) {
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	commentRepo := new(mockCommentRepo)

	svc := NewCommentService(commentRepo, postRepo)
	_, err := svc.Create(context.Background(), 999, 10, "内容")

	assert.ErrorIs(t, err, ErrPostNotFound)
	commentRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCommentService_Create_RepoFails(t *testing.T) {
	postRepo := new(mockPostRepo)
	postRepo.On("FindByID", mock.Anything, uint(1)).Return(&model.Post{ID: 1}, nil)

	commentRepo := new(mockCommentRepo)
	commentRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Comment")).Return(errors.New("db down"))

	svc := NewCommentService(commentRepo, postRepo)
	_, err := svc.Create(context.Background(), 1, 10, "内容")

	require.Error(t, err)
}

func TestCommentService_Delete_ByOwner(t *testing.T) {
	comment := &model.Comment{ID: 100, PostID: 1, UserID: 10}
	commentRepo := new(mockCommentRepo)
	commentRepo.On("FindByID", mock.Anything, uint(100)).Return(comment, nil)
	commentRepo.On("Delete", mock.Anything, uint(100)).Return(nil)

	svc := NewCommentService(commentRepo, new(mockPostRepo))
	err := svc.Delete(context.Background(), 100, 10, model.RoleUser)

	require.NoError(t, err)
}

func TestCommentService_Delete_ByAdmin(t *testing.T) {
	comment := &model.Comment{ID: 100, PostID: 1, UserID: 10}
	commentRepo := new(mockCommentRepo)
	commentRepo.On("FindByID", mock.Anything, uint(100)).Return(comment, nil)
	commentRepo.On("Delete", mock.Anything, uint(100)).Return(nil)

	svc := NewCommentService(commentRepo, new(mockPostRepo))
	err := svc.Delete(context.Background(), 100, 999, model.RoleAdmin)

	require.NoError(t, err)
}

func TestCommentService_Delete_Forbidden(t *testing.T) {
	comment := &model.Comment{ID: 100, PostID: 1, UserID: 10}
	commentRepo := new(mockCommentRepo)
	commentRepo.On("FindByID", mock.Anything, uint(100)).Return(comment, nil)

	svc := NewCommentService(commentRepo, new(mockPostRepo))
	err := svc.Delete(context.Background(), 100, 999, model.RoleUser)

	assert.ErrorIs(t, err, apperrors.ErrForbidden)
	commentRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestCommentService_Delete_NotFound(t *testing.T) {
	commentRepo := new(mockCommentRepo)
	commentRepo.On("FindByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	svc := NewCommentService(commentRepo, new(mockPostRepo))
	err := svc.Delete(context.Background(), 999, 10, model.RoleUser)

	assert.ErrorIs(t, err, ErrCommentNotFound)
}

func TestCommentService_ListByPostID(t *testing.T) {
	commentRepo := new(mockCommentRepo)
	comments := []*model.Comment{{ID: 1}, {ID: 2}}
	commentRepo.On("ListByPostID", mock.Anything, uint(1), 1, 10).Return(comments, int64(2), nil)

	svc := NewCommentService(commentRepo, new(mockPostRepo))
	got, total, err := svc.ListByPostID(context.Background(), 1, 1, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, got, 2)
}
