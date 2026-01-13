# CYP-Registry 部署指南

## 快速开始

### Docker 部署（推荐）

```bash
# 拉取镜像
docker pull cyp-registry:latest

# 运行容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-data:/data \
  -e JWT_SECRET=your-secret-key \
  cyp-registry:latest
```

### Docker Compose 部署

```bash
# 克隆仓库
git clone https://github.com/CYP/registry.git
cd registry

# 启动服务
docker-compose up -d
```

### Kubernetes 部署

```bash
# 应用配置
kubectl apply -f k8s-deployment.yaml

# 检查状态
kubectl get pods -n cyp-registry
```

## 环境配置

### 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| JWT_SECRET | JWT 签名密钥 | 必填 |
| ADMIN_PASSWORD | 管理员初始密码 | admin123 |
| PORT | 服务端口 | 8080 |
| LOG_LEVEL | 日志级别 | info |

### 配置文件

配置文件位于 `configs/config.yaml`，支持热加载。

## 存储配置

### 本地存储

```yaml
storage:
  blob_path: "/data/blobs"
  meta_path: "/data/meta"
  cache_path: "/data/cache"
```

### 云存储（可选）

支持 AWS S3、阿里云 OSS 等对象存储。

## 安全配置

首次部署后：

1. 访问 `http://localhost:8080`
2. 使用默认账号登录：admin / admin123
3. **立即修改默认密码**
4. 配置安全策略

## 高可用部署

### 多副本部署

```yaml
# k8s-deployment.yaml
spec:
  replicas: 3
```

### 负载均衡

使用 Nginx 或云负载均衡器分发流量。

## 监控与告警

### Prometheus 指标

```bash
curl http://localhost:8080/metrics
```

### 健康检查

```bash
curl http://localhost:8080/health
```

## 备份与恢复

### 备份

```bash
# 备份数据目录
tar -czvf backup.tar.gz /data

# 或使用 CLI
./cyp-registry cli backup --output backup.tar.gz
```

### 恢复

```bash
# 恢复数据
tar -xzvf backup.tar.gz -C /data

# 重启服务
docker restart cyp-registry
```

## 升级

```bash
# 拉取新版本
docker pull cyp-registry:latest

# 停止旧容器
docker stop cyp-registry

# 启动新容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-data:/data \
  cyp-registry:latest
```

## 故障排查

### 查看日志

```bash
docker logs cyp-registry
```

### 常见问题

1. **无法登录**: 检查 JWT_SECRET 配置
2. **系统锁定**: 使用 `./scripts/unlock.sh` 解锁
3. **存储空间不足**: 运行清理任务或扩容

## 联系支持

- 文档: https://docs.cyp-registry.com
- 邮箱: nasDSSCYP@outlook.com
- GitHub: https://github.com/CYP/registry

---

**版本**: v1.0.0  
**最后更新**: 2026-01-13
