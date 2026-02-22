# TG WalletBot

一个基于 Telegram 的加密货币钱包机器人，支持 USDT/TRX 充值、提现、转账、红包、收款等功能。

> ⚠️ **声明**: 本项目仅供学习和研究使用，不建议未经安全审计直接用于生产环境。

## ✨ 功能特性

- 💰 **多币种钱包** — 支持 USDT (TRC20)、TRX、CNY
- 📥 **链上充值** — 自动生成充值地址，gRPC 区块扫描实时到账
- 📤 **链上提现** — 支持自动/手动提现，热钱包自动发送
- 🔄 **内部转账** — 零手续费即时到账
- 🧧 **红包** — 支持普通/随机红包，Inline 模式群聊发送
- 💸 **Inline 收款** — @bot 金额 发起收款，深度链接 PIN 验证支付
- 💱 **闪兑** — 多币种实时汇率兑换
- 📈 **余额宝** — 活期理财，每日派息
- 🏪 **商户系统** — API 密钥管理、IP 白名单、回调通知
- 🔐 **安全** — 4 位 PIN 支付密码，免密额度设置
- 🖥️ **管理后台** — React + Ant Design 管理界面

## 🏗️ 技术架构

```
六边形架构 (Hexagonal Architecture)

┌─────────────────────────────────────────┐
│              Adapter Layer              │
│  ┌──────────┐ ┌──────────┐ ┌─────────┐ │
│  │ Telegram  │ │ HTTP API │ │  Admin  │ │
│  │ Handler   │ │ Handler  │ │   Web   │ │
│  └────┬─────┘ └────┬─────┘ └────┬────┘ │
│       └─────────┬───┘────────────┘      │
├─────────────────┼───────────────────────┤
│           App Layer (Use Cases)         │
├─────────────────┼───────────────────────┤
│          Domain Layer (Core)            │
│  ┌────────┐ ┌───────┐ ┌──────────────┐ │
│  │ Entity │ │ Port  │ │   Service    │ │
│  └────────┘ └───────┘ └──────────────┘ │
├─────────────────┼───────────────────────┤
│       Infrastructure Layer              │
│  ┌──────┐ ┌───────┐ ┌──────┐ ┌──────┐ │
│  │MySQL │ │ Redis │ │ TRON │ │Docker│ │
│  └──────┘ └───────┘ └──────┘ └──────┘ │
└─────────────────────────────────────────┘
```

## 🛠️ 技术栈

| 组件 | 技术 |
|------|------|
| 后端语言 | Go 1.22+ |
| Telegram SDK | [go-telegram/bot](https://github.com/go-telegram/bot) |
| HTTP 框架 | Gin |
| ORM | GORM + MySQL 8.0 |
| 缓存 | Redis 7 |
| 区块链 | TRON (gRPC + REST API) |
| 前端 | React 19 + TypeScript + Ant Design 5 |
| 部署 | Docker + Docker Compose + Nginx |

## 🚀 快速开始

### 环境要求

- Go 1.22+
- MySQL 8.0+
- Redis 7+
- Node.js 18+ (前端)

### 1. 克隆项目

```bash
git clone https://github.com/TGlimmer/TG_walletbot.git
cd TG_walletbot
```

### 2. 配置

```bash
cp configs/config.yaml.example configs/config.yaml
# 编辑 configs/config.yaml，填入你的 Bot Token、数据库信息等
```

### 3. 数据库迁移

```bash
go run cmd/migrate/main.go -config configs/config.yaml -direction up
```

### 4. 启动 Bot

```bash
go run cmd/bot/main.go -config configs/config.yaml
```

### 5. 启动管理后台 API

```bash
go run cmd/api/main.go -config configs/config.yaml
```

### 6. 启动前端 (开发模式)

```bash
cd web && npm install && npm run dev
```

### Docker Compose 部署

```bash
cd deployments && docker-compose up -d
```

## 📁 项目结构

```
TG_walletbot/
├── cmd/
│   ├── bot/          # Telegram Bot 入口
│   ├── api/          # Admin API 入口
│   └── migrate/      # 数据库迁移工具
├── internal/
│   ├── domain/       # 领域层 (entity/port/service)
│   ├── adapter/      # 适配器层 (handler/repository/telegram)
│   ├── app/          # 应用层 (use case)
│   └── infrastructure/ # 基础设施 (config/db/tron)
├── pkg/              # 公共工具包
├── web/              # React 前端
├── migrations/       # SQL 迁移文件
├── deployments/      # Docker 部署配置
└── configs/          # 配置模板
```

## 🏆 打赏榜单

感谢以下用户的支持与赞助！

<!-- 打赏榜单将在此更新 -->

| 排名 | 用户 | 金额 |
|------|------|------|
| 🥇 | [小谢](https://t.me/xiaoxie_dad) | 18.8 USDT |
| 🥈 | — | — |
| 🥉 | — | — |

## 💬 技术交流 / 意见反馈

如有任何问题或建议，欢迎提交 [Issue](https://github.com/TGlimmer/TG_walletbot/issues)。

- **MCG 技术交流群**: [https://t.me/MCG_Club](https://t.me/MCG_Club)

## 🔧 私有定制

如需定制机器人或其他业务，请联系 [@Miya](https://t.me/Miya)

## 📄 License

[MIT License](LICENSE)

## 🤝 Contributing

欢迎提交 Issue 和 Pull Request。

提交代码前请确保：
1. `go build ./...` 无错误
2. `go vet ./...` 无警告
3. `go test ./...` 全部通过
