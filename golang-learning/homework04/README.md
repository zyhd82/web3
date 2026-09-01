# 个人博客后端服务

Go + Gin + GORM 实现的个人博客后端微服务：用户认证（JWT）、用户管理、文章管理、评论管理，配 Swagger 文档、统一错误处理和结构化日志。

## 技术栈

- Web 框架：[Gin](https://github.com/gin-gonic/gin)
- ORM：[GORM](https://gorm.io/) + MySQL 8.x
- 鉴权：JWT（[golang-jwt/jwt](https://github.com/golang-jwt/jwt)）+ bcrypt
- API 文档：[swaggo](https://github.com/swaggo/swag) + gin-swagger
- 日志：[logrus](https://github.com/sirupsen/logrus)
- 测试：[testify](https://github.com/stretchr/testify)（mock + assert）

## 快速开始

```bash
cp .env.example .env   # 按实际情况改数据库连接、JWT_SECRET
go run ./cmd/server
```

服务启动后：
- 健康检查：`GET /healthz`
- API 前缀：`/api/v1`
- Swagger UI：`http://localhost:8080/swagger/index.html`（右上角 Authorize 输入 `Bearer <登录拿到的 token>` 可以直接在页面上调试需要登录的接口）

首次启动会自动 `AutoMigrate` 建表（`users`/`posts`/`comments`），不需要手动建库建表。

## 开发

```bash
go build ./...       # 编译
go vet ./...          # 静态检查
go test ./...          # 跑全部单元测试（不连数据库）
```

改了 handler 的 Swagger 注释后，记得重新生成文档并把 `docs/` 的改动一起提交：

```bash
swag init -g cmd/server/main.go -o docs
```

## API 概览

| 方法   | 路径                          | 说明               | 需要登录 |
| ------ | ----------------------------- | ------------------ | -------- |
| POST   | `/api/v1/auth/register`       | 注册               | 否       |
| POST   | `/api/v1/auth/login`          | 登录，返回 JWT      | 否       |
| GET    | `/api/v1/users/me`            | 当前用户资料        | 是       |
| PUT    | `/api/v1/users/me`            | 更新资料            | 是       |
| PUT    | `/api/v1/users/me/password`   | 修改密码            | 是       |
| GET    | `/api/v1/posts`               | 文章列表（分页）     | 否       |
| GET    | `/api/v1/posts/{postId}`      | 文章详情            | 否       |
| POST   | `/api/v1/posts`               | 创建文章            | 是       |
| PUT    | `/api/v1/posts/{postId}`      | 更新文章（作者/管理员）| 是     |
| DELETE | `/api/v1/posts/{postId}`      | 删除文章（作者/管理员）| 是     |
| GET    | `/api/v1/posts/{postId}/comments` | 评论列表（分页） | 否       |
| POST   | `/api/v1/posts/{postId}/comments` | 创建评论         | 是       |
| DELETE | `/api/v1/comments/{id}`       | 删除评论（作者/管理员）| 是     |

完整请求/响应结构以 Swagger 文档为准。所有响应统一为 `{ "code": int, "message": string, "data": any }`，`code=0` 表示成功。
