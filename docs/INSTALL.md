# 安装文档

CYP-Docker-Registry 安装和配置指南

**作者：** CYP | **联系方式：** nasDSSCYP@outlook.com

## 目录

- [系统要求](#系统要求)
- [快速开始](#快速开始)
- [Docker 部署](#docker-部署)
- [手动部署](#手动部署)
- [配置说明](#配置说明)
- [Docker 客户端配置](#docker-客户端配置)
- [常见问题](#常见问题)

---

## 系统要求

### 硬件要求

| 项目 | 最低配置 | 推荐配置 |
|------|---------|---------|
| CPU | 1 核 | 2 核以上 |
| 内存 | 512 MB | 2 GB 以上 |
| 磁盘 | 10 GB | 100 GB 以上 |

### 软件要求

- **操作系统**: Linux (推荐)、macOS、Windows
- **Docker**: 20.10+ (Docker 部署)
- **Docker Compose**: 2.0+ (Docker 部署)
- **Go**: 1.21+ (手动编译)
- **Node.js**: 20+ (手动编译前端)

---

## 快速开始

### 使用 Docker Compose（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/CYP/cyp-docker-registry.git
cd cyp-docker-registry

# 2. 复制配置文件
cp configs/config.yaml.example configs/config.yaml

# 3. 启动服务
docker-compose up -d

# 4. 查看日志
docker-compose logs -f

# 5. 访问 Web 界面
# 打开浏览器访问 http://localhost:8080
```

### 验证安装

```bash
# 检查服务健康状态
curl http://localhost:8080/health

# 检查版本信息
curl http://localhost:8080/api/version

# 测试 Docker Registry V2 API
curl http://localhost:8080/v2/
```

---

## Docker 部署

### 使用 Docker Compose

1. **创建配置文件**

```bash
mkdir -p configs
cp configs/config.yaml.example configs/config.yaml
```

2. **编辑配置文件**（可选）

```bash
vim configs/config.yaml
```

3. **启动服务**

```bash
docker-compose up -d
```

4. **管理服务**

```bash
# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看日志
docker-compose logs -f

# 重新构建并启动
docker-compose up -d --build
```

### 使用 Docker 命令

```bash
# 构建镜像
docker build -t cyp-docker-registry:latest .

# 创建数据目录
mkdir -p ./data/blobs ./data/meta ./data/cache

# 运行容器
docker run -d \
  --name cyp-docker-registry \
  -p 8080:8080 \
  -v $(pwd)/data/blobs:/app/data/blobs \
  -v $(pwd)/data/meta:/app/data/meta \
  -v $(pwd)/data/cache:/app/data/cache \
  -v $(pwd)/configs:/app/configs:ro \
  --restart unless-stopped \
  cyp-docker-registry:latest
```

### 数据持久化

Docker 部署使用以下卷存储数据：

| 卷名称 | 容器路径 | 说明 |
|-------|---------|------|
| cyp-docker-registry-blobs | /app/data/blobs | 镜像层数据 |
| cyp-docker-registry-meta | /app/data/meta | 元数据和凭证 |
| cyp-docker-registry-cache | /app/data/cache | 加速器缓存 |

**备份数据：**

```bash
# 备份所有数据
docker run --rm \
  -v cyp-docker-registry-blobs:/blobs \
  -v cyp-docker-registry-meta:/meta \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/registry-backup.tar.gz /blobs /meta
```

---

## 手动部署

### 编译后端

```bash
# 安装 Go 依赖
go mod download

# 编译
CGO_ENABLED=0 go build -o server ./cmd/server

# 运行
./server -config ./configs/config.yaml
```

### 编译前端

```bash
cd web

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
```

### 目录结构

```
cyp-docker-registry/
├── server              # 后端可执行文件
├── VERSION             # 版本号文件
├── configs/
│   └── config.yaml     # 配置文件
├── data/
│   ├── blobs/          # 镜像层存储
│   ├── meta/           # 元数据存储
│   └── cache/          # 缓存存储
└── web/
    └── dist/           # 前端构建产物
```

---

## 配置说明

### 配置文件位置

- Docker 部署: `/app/configs/config.yaml`
- 手动部署: `./configs/config.yaml`

### 主要配置项

#### 服务器配置

```yaml
server:
  port: 8080              # 监听端口
  host: "0.0.0.0"         # 监听地址
  timeout: 30             # 请求超时（秒）
  max_body_size: "1GB"    # 最大请求体大小
```

#### 存储配置

```yaml
storage:
  blob_path: "./data/blobs"     # 镜像层存储路径
  meta_path: "./data/meta"      # 元数据存储路径
  cache_path: "./data/cache"    # 缓存存储路径
  max_cache_size: "10GB"        # 最大缓存大小
```

#### 加速器配置

```yaml
accelerator:
  enabled: true           # 启用加速器
  upstreams:              # 上游源列表
    - name: "Docker Hub"
      url: "https://registry-1.docker.io"
      priority: 1
      enabled: true
    - name: "阿里云镜像"
      url: "https://registry.cn-hangzhou.aliyuncs.com"
      priority: 2
      enabled: true
```

#### 更新配置

```yaml
update:
  check_interval: "24h"   # 检查更新间隔
  auto_update: false      # 自动更新
  update_url: "https://api.github.com/repos/CYP/cyp-docker-registry/releases/latest"
```

#### 认证配置

```yaml
auth:
  enabled: false          # 启用认证
  username: ""            # 用户名
  password: ""            # 密码
```

---

## Docker 客户端配置

### 配置 Docker 信任本地仓库

由于本地仓库默认使用 HTTP，需要配置 Docker 信任该仓库。

#### Linux

编辑 `/etc/docker/daemon.json`：

```json
{
  "insecure-registries": ["localhost:8080", "your-server-ip:8080"]
}
```

重启 Docker：

```bash
sudo systemctl restart docker
```

#### macOS / Windows (Docker Desktop)

1. 打开 Docker Desktop 设置
2. 进入 "Docker Engine" 选项
3. 添加 `insecure-registries` 配置
4. 点击 "Apply & Restart"

### 使用仓库

```bash
# 登录（如果启用了认证）
docker login localhost:8080

# 标记镜像
docker tag myapp:latest localhost:8080/myapp:latest

# 推送镜像
docker push localhost:8080/myapp:latest

# 拉取镜像
docker pull localhost:8080/myapp:latest

# 列出镜像
curl http://localhost:8080/api/images
```

### 使用加速器拉取公共镜像

```bash
# 通过加速器拉取 Docker Hub 镜像
docker pull localhost:8080/library/nginx:latest

# 通过加速器拉取其他仓库镜像
docker pull localhost:8080/username/image:tag
```

---

## 常见问题

### Q: 无法推送镜像，提示 "http: server gave HTTP response to HTTPS client"

**A:** Docker 默认使用 HTTPS，需要配置信任本地 HTTP 仓库。参见 [Docker 客户端配置](#docker-客户端配置)。

### Q: 容器启动失败，提示权限错误

**A:** 检查数据目录权限：

```bash
# Docker 部署
docker-compose down
docker volume rm cyp-docker-registry-blobs cyp-docker-registry-meta cyp-docker-registry-cache
docker-compose up -d

# 手动部署
chmod -R 755 ./data
```

### Q: 如何修改监听端口？

**A:** 修改配置文件和 Docker 端口映射：

1. 编辑 `configs/config.yaml`，修改 `server.port`
2. 修改 `docker-compose.yaml` 中的端口映射

### Q: 如何清理缓存？

**A:** 通过 API 或 Web 界面清理：

```bash
# API 方式
curl -X DELETE http://localhost:8080/api/accel/cache

# 或直接删除缓存目录
rm -rf ./data/cache/*
```

### Q: 如何备份和恢复数据？

**A:** 备份 `data` 目录即可：

```bash
# 备份
tar czf registry-backup.tar.gz ./data

# 恢复
tar xzf registry-backup.tar.gz
```

### Q: 如何启用 HTTPS？

**A:** 建议使用反向代理（如 Nginx）处理 HTTPS：

```nginx
server {
    listen 443 ssl;
    server_name registry.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        client_max_body_size 0;
    }
}
```

---

## 技术支持

如有问题，请联系：

- **作者**: CYP
- **邮箱**: nasDSSCYP@outlook.com

---

## 版权声明

Copyright © 2026 CYP. All rights reserved.
