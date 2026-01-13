# CYP-Registry

零信任架构的企业级容器镜像私有仓库管理系统

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](VERSION)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](docs/LICENSE.md)

## 特性

- 🔐 **零信任安全** - 强制登录认证，入侵检测，自动锁定
- 🚀 **智能加速** - P2P 分发，多源镜像，智能缓存
- 📦 **供应链安全** - 镜像签名，SBOM 生成，漏洞扫描
- 🏢 **团队协作** - 组织管理，RBAC 权限，分享链接
- 🌍 **全平台支持** - Docker/K8s/NAS/树莓派/云环境
- 📊 **审计追踪** - 区块链哈希防篡改，完整审计日志

## 快速开始

### Docker 部署

```bash
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-data:/data \
  -e JWT_SECRET=your-secret-key \
  cyp-registry:latest
```

### Docker Compose

```bash
git clone https://github.com/CYP/cyp-registry.git
cd cyp-registry
docker-compose up -d
```

访问 http://localhost:8080，使用默认账号登录：
- 用户名: `admin`
- 密码: `admin123`

⚠️ **首次登录后请立即修改默认密码！**

## 安全特性

- 登录失败 3 次自动锁定系统
- 所有页面必须登录后访问
- 审计日志使用区块链哈希防篡改
- 支持 IP 绑定和地理位置检测

## 文档

- [部署指南](docs/DEPLOY.md)
- [安全指南](docs/SECURITY.md)
- [API 文档](docs/API.md)
- [安装说明](docs/INSTALL.md)

## 构建

```bash
# 安装依赖
make deps

# 构建
make build

# 运行测试
make test
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.21+ |
| 前端 | Vue 3 + Vite + Element Plus |
| 数据库 | SQLite |
| 容器 | Docker / Kubernetes |

## 许可证

MIT License - 详见 [LICENSE](docs/LICENSE.md)

## 联系方式

- 作者: CYP
- 邮箱: nasDSSCYP@outlook.com
- GitHub: https://github.com/CYP/cyp-registry

---

**版本**: v1.0.0 | **最后更新**: 2026-01-13
