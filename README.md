# outsideapi

这是一个轻量的 Go 服务仓库，提供若干与第三方平台交互的 HTTP API（示例包含 PayPal 与 Facebook OAuth）。

**模块路径:** `github.com/nkypy/outsideapi`

**要求**
- Go 1.25+

**快速开始**

1. 获取依赖并构建：

```bash
go mod tidy
go build ./...
```

2. 运行（默认端口 `8080`）：

```bash
go run cmd/server/main.go
# 或者
PORT=9090 go run cmd/server/main.go
```

**环境变量**
- `PORT`：服务监听端口，默认 `8080`。
- `PAYPAL_CALLBACK_URL`：当 PayPal 回调发生时，服务将把回调数据转发到该 URL（见 `paypal.go`）。
- `FACEBOOK_CALLBACK_URL`：当 Facebook OAuth 回调完成后，服务会把信息转发到该 URL（见 `facebook.go`）。

（如果你使用 `.env`，仓库中已包含 `github.com/joho/godotenv`，可自行在 `main` 中加载）

**主要路由**

- PayPal 相关（由 `PaypalRouter` 注册到 `/pp`）：
  - `POST /pp/orders` - 创建订单
  - `POST /pp/orders/:id/capture` - 捕获订单
  - `GET  /pp/orders/:id` - 获取订单详情
  - `GET  /pp/captures/:id` - 获取捕获详情
  - `POST /pp/captures/:id/refund` - 退款
  - `POST /pp/callback` - 接收 PayPal webhook 并转发

- Facebook 相关（由 `FacebookRouter` 注册到 `/fb`）：
  - `GET  /fb/login` - 生成登录 URL
  - `POST /fb/callback` - 接收 OAuth 回调并转发

（项目里以 `outsideapi.PaypalRouter` 和 `outsideapi.FacebookRouter` 的形式导出路由注册函数，可在 `main` 中创建 `gin.Engine` 并调用这些函数进行挂载。）

**示例请求**

创建订单示例：

```bash
curl -X POST http://localhost:8080/pp/orders \
  -H 'Content-Type: application/json' \
  -d '{"purchase_units": [{"amount": {"currency_code": "USD", "value": "1.00"}}]}'
```

Facebook 登录 URL：

```bash
curl http://localhost:8080/fb/login
```

**贡献**
- 欢迎提交 Issue 与 Pull Request。请保持小改动且附带描述。

**许可证**
本仓库使用 MIT 许可证 — 见 `LICENSE` 文件。
