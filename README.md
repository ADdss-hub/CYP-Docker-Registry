# CYP-Docker-Registry

零信任容器镜像仓库，让安全与高效兼得

[![Version](https://img.shields.io/badge/version-1.2.3-blue.svg)](VERSION)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](docs/LICENSE.md)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org)
[![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D.svg)](https://vuejs.org)

专为个人开发者与小型团队打造的企业级容器镜像管理解决方案。基于零信任架构重构，融合供应链安全、智能加速、全场景适配能力，无需复杂配置，开箱即享安全可控的镜像管理体验。

## 核心亮点

### 🔐 零信任安全防护
- 强制登录认证贯穿全链路，API/界面/资源均需身份校验
- 3次登录失败、5次令牌错误或10次未授权API访问即触发自动锁定
- 内置入侵检测引擎（IDS），精准识别伪造JWT、IP异常变更、绕过登录等风险行为
- 硬件资源限制、网络封锁、服务暂停三重锁定机制
- 区块链哈希审计日志不可篡改，1年数据留存

### 🚀 极致传输性能
- P2P去中心化分发 + 分层压缩（zstd/gzip自适应）+ 智能缓存三重加速
- NAT自动穿透打破网络壁垒，速度提升50-80%
- 支持本地/云/混合存储智能切换

### 📦 全链路供应链安全
- 自动生成SBOM软件物料清单（Syft）
- 集成Trivy漏洞扫描，实时预警CRITICAL/HIGH级风险
- Sigstore/cosign强制镜像签名，杜绝恶意篡改

### 🌍 全平台适配
- 从X86到ARM，从树莓派、NAS到K8s、云服务器，一键启动
- 内置环境感知引擎，自动识别Docker/阿里云/群晖/物理机等运行场景
- 动态优化配置，零配置体验

### 🏢 团队协作
- 组织命名空间 + RBAC权限控制
- 密码保护分享链接
- 个人访问令牌管理

## 快速开始

### Docker 部署（推荐）

```bash
docker run -d \
  --name cyp-docker-registry \
  -p 8080:8080 \
  -v cyp-data:/data \
  -e JWT_SECRET=your-secret-key \
  cyp-docker-registry:latest
```

### Docker Compose

```bash
git clone https://github.com/CYP/cyp-docker-registry.git
cd cyp-docker-registry
docker-compose up -d
```

访问 http://localhost:8080，使用默认账号登录：
- 用户名: `admin`
- 密码: `admin123`

⚠️ **首次登录后请立即修改默认密码！**

## 适用场景

- **个人开发者**：安全存储个人项目镜像，免担心泄露风险
- **小型团队**：低成本实现团队镜像共享、版本管控与权限隔离
- **边缘部署**：适配树莓派等低功耗设备，满足边缘计算场景需求
- **合规需求**：符合GDPR/HIPAA/SOC2合规基线，审计日志可追溯

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.21+ |
| 前端 | Vue 3 + Vite + Element Plus |
| 数据库 | SQLite（零配置） |
| 安全组件 | Sigstore/cosign、Syft SBOM、Trivy |
| 传输优化 | libp2p P2P引擎、Redis缓存 |
| 部署方式 | Docker / Docker Compose / Kubernetes |

## 文档

- [部署指南](docs/DEPLOY.md)
- [安全指南](docs/SECURITY.md)
- [API 文档](docs/API.md)
- [安装说明](docs/INSTALL.md)
- [更新日志](docs/CHANGELOG.md)

## 构建

```bash
# 安装依赖
make deps

# 构建
make build

# 运行测试
make test
```

## 许可证

MIT License - 详见 [LICENSE](docs/LICENSE.md)

## 联系方式

- 📌 GitHub: https://github.com/CYP/cyp-docker-registry
- 📚 官方文档: https://docs.cyp-docker-registry.com
- 📧 技术支持: nasDSSCYP@outlook.com
- 🔒 安全反馈: security@cyp-docker-registry.com

---

**版本**: v1.2.3 | **最后更新**: 2026-01-15

