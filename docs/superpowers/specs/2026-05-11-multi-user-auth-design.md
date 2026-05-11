# 多用户账号体系 — 设计文档

## 1. 概述

在现有单用户股票盯盘工具基础上，新增多用户账号体系。用户通过邀请码注册，JWT 认证，所有数据按用户隔离。

**破坏性变更：** 现有 SQLite 数据库需要迁移，所有业务表新增 `user_id` 字段。

---

## 2. 数据模型

### 2.1 新增表

```sql
CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,          -- bcrypt hash (cost=12)
    role       TEXT DEFAULT 'user',    -- 'admin' | 'user'
    created_at TEXT NOT NULL           -- ISO 8601
);

CREATE TABLE invite_codes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    code       TEXT NOT NULL UNIQUE,
    max_uses   INTEGER DEFAULT 1,      -- 0 = 无限
    used_count INTEGER DEFAULT 0,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL,
    is_active  INTEGER DEFAULT 1
);
```

### 2.2 现有表迁移

```sql
ALTER TABLE watchlist ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0 REFERENCES users(id);
ALTER TABLE holdings  ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0 REFERENCES users(id);
ALTER TABLE alerts    ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0 REFERENCES users(id);
```

- `DEFAULT 0` 用于迁移已有数据到默认管理员
- 首次启动时自动创建 admin 用户（密码随机生成，通过日志输出）

---

## 3. 认证方案

### 3.1 JWT

- 签名算法：HS256
- Secret：`.env` 中 `JWT_SECRET`；启动时如果未配置则自动生成随机 64 字符 secret 兜底
- 有效期：7 天（`exp = iat + 604800`）
- Payload：`{ user_id, username, role, exp, iat }`

### 3.2 密码加密

- bcrypt，`cost = 12`
- 使用已存在的 `golang.org/x/crypto/bcrypt` 依赖，不引入新库

### 3.3 依赖

- 新增 `github.com/golang-jwt/jwt/v5` — JWT 签发和校验

---

## 4. API 路由

### 4.1 无需认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/register` | 注册 `{ username, password, inviteCode }` |
| POST | `/api/auth/login` | 登录 `{ username, password }` → `{ token, user }` |

### 4.2 需管理员

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/invite-codes` | 生成邀请码 `{ maxUses, count? }` |
| GET | `/api/admin/invite-codes` | 邀请码列表 |
| PUT | `/api/admin/invite-codes/:id` | 禁用/启用 `{ isActive }` |

### 4.3 需 JWT 认证（现有路由全部加上中间件）

所有 `/api/*` 现有路由（watchlist、alerts、holdings、quote、kline）统一经过 JWT 中间件。中间件从 `Authorization: Bearer <token>` 提取 token，校验后将 `user_id`、`username`、`role` 注入 Gin context。

**WebSocket 认证：**
连接地址 `ws://host/ws?token=<jwt>`，handler 从 query 参数校验 JWT，失败返回 401 并关闭连接。

---

## 5. 前端改造

### 5.1 Web 前端

| 文件 | 改动 |
|------|------|
| `web/login.html` | 新增：登录/注册双表单页 |
| `web/js/auth.js` | 新增：登录、注册、token 管理 |
| `web/js/api.js` | 改造：所有请求加 `Authorization` header，401 跳转登录 |
| `web/js/app.js` | 改造：启动时检查 token，无 token 跳转登录页 |
| `web/js/admin.js` | 新增：管理员邀请码管理界面 |
| `web/admin.html` | 新增：管理页面 |

**登录页 UI：**
- 默认显示登录表单（用户名 + 密码 + 登录按钮）
- 底部"没有账号？注册"链接切换到注册表单（用户名 + 密码 + 确认密码 + 邀请码 + 注册按钮）
- 登录成功后 token 存入 `localStorage`，跳转主页

### 5.2 Flutter 移动端

| 文件 | 改动 |
|------|------|
| `lib/presentation/screens/login_screen.dart` | 新增：登录/注册页 |
| `lib/presentation/providers/auth_provider.dart` | 新增：登录状态管理 |
| `lib/data/api/api_client.dart` | 改造：Dio 拦截器注入 token，401 自动跳转 |
| `lib/app.dart` | 改造：路由守卫，未登录 → LoginScreen |

### 5.3 数据迁移

`cmd/migrate/main.go` 新增：
1. 执行 `ALTER TABLE ... ADD user_id` DDL
2. 创建默认 admin 用户
3. 将已有数据 `user_id` 设为 admin 的 id

---

## 6. 中间件设计

```go
// internal/middleware/auth.go
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
    // 1. 提取 Authorization header
    // 2. 解析 JWT
    // 3. 校验签名 + 过期
    // 4. 注入 c.Set("user_id", ...) c.Set("username", ...) c.Set("role", ...)
    // 5. c.Next()
}

func AdminRequired() gin.HandlerFunc {
    // 检查 c.GetString("role") == "admin"，否则 403
}
```

路由注册：
```go
api := r.Group("/api")
api.POST("/auth/register", authH.Register)
api.POST("/auth/login", authH.Login)

auth := api.Group("", middleware.AuthMiddleware(cfg.JwtSecret))
// 所有现有 handler 注册到 auth group 下
watchlistH.Register(auth)
// ...

admin := auth.Group("/admin", middleware.AdminRequired())
admin.GET("/invite-codes", adminH.ListCodes)
// ...
```

---

## 7. 不做的

- 不支持 OAuth2 / 第三方登录
- 不支持密码重置（手动联系管理员改密码）
- 不支持邮箱验证
- 不引入 redis / 外部 session 存储
- 不做请求频率限制（rate limiting），后续再考虑

---

## 8. 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/model/user.go` | 创建 | User、InviteCode 模型 |
| `internal/db/sqlite.go` | 修改 | 新增 users、invite_codes DDL + 迁移 ALTER |
| `internal/config/config.go` | 修改 | 新增 JWT_SECRET、首次启动 ADMIN 密码 |
| `internal/middleware/auth.go` | 创建 | JWT 中间件 + AdminRequired |
| `internal/repo/user.go` | 创建 | User CRUD + InviteCode CRUD |
| `internal/handler/auth.go` | 创建 | register、login handler |
| `internal/handler/admin.go` | 创建 | invite-codes CRUD handler |
| `internal/repo/watchlist.go` | 修改 | 所有查询加 user_id |
| `internal/repo/alert.go` | 修改 | 所有查询加 user_id |
| `internal/repo/holding.go` | 修改 | 所有查询加 user_id |
| `internal/ws/hub.go` | 修改 | /ws 加 JWT 校验 |
| `cmd/server/main.go` | 修改 | 注册授权路由 + 中间件 + admin 初始化 |
| `cmd/migrate/main.go` | 修改 | 数据迁移脚本 |
| `web/login.html` | 创建 | 登录/注册页 |
| `web/admin.html` | 创建 | 管理页面 |
| `web/js/auth.js` | 创建 | 登录/注册逻辑 |
| `web/js/admin.js` | 创建 | 管理页面逻辑 |
| `web/js/api.js` | 修改 | 加 Authorization header |
| `web/js/app.js` | 修改 | 启动 token 检查 |
| `go.mod` | 修改 | 新增 golang-jwt 依赖 |
| Flutter 相关文件 | 修改 | login_screen、auth_provider、api_client、app.dart |

---

## 9. 初始化流程

首次启动时：
1. 检测 `users` 表是否为空
2. 如果为空，自动创建 admin 用户（密码随机生成，打印到日志）
3. 自动生成 1 个初始邀请码（`max_uses = 1`），也打印到日志

管理员用日志中的密码登录后，可以：
- 修改自己的密码
- 生成更多邀请码
- 管理其他用户
