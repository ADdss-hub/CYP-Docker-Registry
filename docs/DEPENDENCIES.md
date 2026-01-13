# 依赖清单

**版本**: v1.0.0  
**更新时间**: 2026-01-13

## Go 依赖 (go.mod)

### 核心依赖

| 包名 | 版本 | 说明 | 兼容性 |
|------|------|------|--------|
| github.com/gin-gonic/gin | v1.10.0 | Web 框架 | Go 1.21+ |
| github.com/golang-jwt/jwt/v5 | v5.2.1 | JWT 认证 | Go 1.18+ |
| github.com/gorilla/websocket | v1.5.3 | WebSocket | Go 1.12+ |
| github.com/mattn/go-sqlite3 | v1.14.22 | SQLite 驱动 | CGO |
| github.com/spf13/viper | v1.19.0 | 配置管理 | Go 1.18+ |
| go.uber.org/zap | v1.27.0 | 日志 | Go 1.19+ |
| golang.org/x/crypto | v0.24.0 | 加密库 | Go 1.18+ |
| gopkg.in/yaml.v3 | v3.0.1 | YAML 解析 | Go 1.15+ |

### P2P 网络依赖

| 包名 | 版本 | 说明 | 备注 |
|------|------|------|------|
| github.com/libp2p/go-libp2p | v0.33.2 | P2P 网络库 | 稳定版本 |
| github.com/libp2p/go-libp2p-kad-dht | v0.25.2 | DHT 实现 | 与 libp2p 0.33 兼容 |
| github.com/multiformats/go-multiaddr | v0.12.4 | 多地址格式 | - |

### 版本选择说明

1. **gin v1.10.0**: 最新稳定版，支持 Go 1.21+
2. **libp2p v0.33.2**: 选择 0.33.x 而非 0.37+ 是因为：
   - 0.37+ 有 breaking changes (core 模块分离)
   - 0.33.x 更稳定，依赖更少
3. **viper v1.19.0**: 最新稳定版，配置热加载支持好

## NPM 依赖 (web/package.json)

### 生产依赖

| 包名 | 版本 | 说明 | 兼容性 |
|------|------|------|--------|
| vue | ^3.5.0 | Vue 3 框架 | Node 18+ |
| vue-router | ^4.4.0 | 路由 | Vue 3 |
| pinia | ^2.2.0 | 状态管理 | Vue 3 |
| element-plus | ^2.8.0 | UI 组件库 | Vue 3.3+ |
| axios | ^1.7.0 | HTTP 客户端 | - |

### 开发依赖

| 包名 | 版本 | 说明 |
|------|------|------|
| vite | ^5.4.0 | 构建工具 |
| typescript | ~5.5.0 | TypeScript |
| vue-tsc | ^2.1.0 | Vue TS 检查 |
| vitest | ^2.0.0 | 测试框架 |
| eslint | ^8.57.0 | 代码检查 |

### 版本选择说明

1. **Vue 3.5.0**: 最新稳定版，性能优化
2. **Element Plus 2.8.0**: 稳定版，组件丰富
3. **Vite 5.4.0**: 最新稳定版，构建速度快
4. **TypeScript 5.5.0**: LTS 版本，稳定性好

## 系统要求

### 后端

- Go 1.21 或更高版本
- CGO 支持 (SQLite 需要)
- 支持的操作系统: Linux, macOS, Windows

### 前端

- Node.js 18.0.0 或更高版本
- npm 9.0.0 或更高版本

## 安全更新策略

1. **定期检查**: 每月检查依赖更新
2. **安全补丁**: 立即更新安全相关补丁
3. **主版本升级**: 评估后谨慎升级
4. **测试验证**: 升级后完整测试

## 更新依赖

```bash
# Go 依赖更新
go get -u ./...
go mod tidy

# NPM 依赖更新
cd web
npm update
npm audit fix
```

## 检查过时依赖

```bash
# Go
go list -u -m all

# NPM
npm outdated
```
