package service

import (
	"context"

	"homework04/internal/model"

	"github.com/stretchr/testify/mock"
)

// mockUserRepo 同时满足 authUserRepository 和 profileRepository（方法是两者的并集），
// 两个 service 的测试共用一个 mock 类型即可。
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *mockUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

// mockPostRepo 满足 postRepository；FindByID 这一个方法同时也满足 commentPostRepository，
// 两处测试共用。
type mockPostRepo struct {
	mock.Mock
}

func (m *mockPostRepo) Create(ctx context.Context, post *model.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

func (m *mockPostRepo) Update(ctx context.Context, post *model.Post) error {
	args := m.Called(ctx, post)
	return args.Error(0)
}

func (m *mockPostRepo) FindByID(ctx context.Context, id uint) (*model.Post, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Post), args.Error(1)
}

func (m *mockPostRepo) List(ctx context.Context, page, pageSize int, userID *uint) ([]*model.Post, int64, error) {
	args := m.Called(ctx, page, pageSize, userID)
	var posts []*model.Post
	if args.Get(0) != nil {
		posts = args.Get(0).([]*model.Post)
	}
	return posts, args.Get(1).(int64), args.Error(2)
}

// mockCommentRepo 满足 commentRepository。
type mockCommentRepo struct {
	mock.Mock
}

func (m *mockCommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}

func (m *mockCommentRepo) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockCommentRepo) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Comment), args.Error(1)
}

func (m *mockCommentRepo) ListByPostID(ctx context.Context, postID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	args := m.Called(ctx, postID, page, pageSize)
	var comments []*model.Comment
	if args.Get(0) != nil {
		comments = args.Get(0).([]*model.Comment)
	}
	return comments, args.Get(1).(int64), args.Error(2)
}
