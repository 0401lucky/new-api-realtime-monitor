# 实时监控台

这是一个可直接部署的 Go 后端 + 单页前端监控台，用于监控 New API 的请求日志、Key 用量和渠道表现。

## 本地运行

```bash
go test ./...
go run ./cmd/server
```

打开 `http://localhost:8080`。

## Zeabur 部署

项目已提供 `Dockerfile`，在 Zeabur 新建服务并指向本仓库即可。

部署步骤：

1. 在 Zeabur 新建 Project。
2. 选择从 GitHub 导入本仓库。
3. 部署方式选择 Dockerfile。
4. 配置下面的 New API 数据库环境变量。
5. 部署完成后绑定域名或直接使用 Zeabur 默认域名访问。

不需要挂载卷。服务本身是无状态的，只读取 New API 数据库并托管前端静态文件。

## 连接 New API 数据库

后端会优先读取 New API 数据库；未配置数据库时，会使用演示数据方便本地预览。

推荐在 Zeabur 中配置：

- `NEW_API_DB_DRIVER`：`mysql` 或 `postgres`
- `NEW_API_DSN`：数据库连接串

MySQL 示例：

```bash
NEW_API_DB_DRIVER=mysql
NEW_API_DSN=user:password@tcp(host:3306)/new-api?charset=utf8mb4&parseTime=true&loc=Local
```

PostgreSQL 示例：

```bash
NEW_API_DB_DRIVER=postgres
NEW_API_DSN=postgres://user:password@host:5432/new_api?sslmode=require
```

如果 Zeabur 已注入 `MYSQL_URL`、`POSTGRES_URL` 或 `DATABASE_URL`，服务会自动识别。

当前读取的 New API 表：

- `logs`：请求、错误、模型、额度、Tokens、耗时统计
- `tokens`：Key 信息和剩余额度
- `users`：用户额度和请求数
- `channels`：渠道状态、权重、响应时间和余额

服务只做读查询，不会写入 New API 数据库。

## 可选环境变量

- `PORT`：服务端口，Zeabur 通常会自动注入
- `SYSTEM_NAME`：站点名称
- `SERVER_ADDRESS`：前端展示的平台地址，默认 `/`
- `DOCS_LINK`：文档链接
- `LOGO_URL`：Logo 地址
- `CACHE_TTL_SECONDS`：前端刷新缓存周期
- `QUOTA_PER_UNIT`：额度换算单位
- `APP_VERSION`：展示版本号
