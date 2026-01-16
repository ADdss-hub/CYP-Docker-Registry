# 镜像保存使用说明

**CYP-Docker-Registry 镜像保存与管理指南**

---

## 一、概述

CYP-Docker-Registry 提供完整的 Docker 镜像保存、管理和分发功能。本文档将指导您如何将镜像推送到私有仓库、从仓库拉取镜像，以及进行镜像的备份与迁移。

---

## 二、前置准备

### 2.1 确认服务运行

```bash
# 检查服务状态
curl http://localhost:8080/health

# 预期响应
{"success":true,"data":{"status":"healthy","version":"1.2.0"}}
```

### 2.2 登录认证

系统采用零信任架构，所有操作需先登录认证。

**Web 界面登录：**
访问 `http://localhost:8080`，使用账号密码登录。

**命令行登录：**
```bash
docker login localhost:8080
# 输入用户名和密码
```

---

## 三、镜像推送（保存到仓库）

### 3.1 标记镜像

将本地镜像标记为仓库地址格式：

```bash
# 格式：docker tag <本地镜像>:<标签> <仓库地址>/<镜像名>:<标签>
docker tag myapp:latest localhost:8080/myapp:latest

# 示例：标记多个版本
docker tag myapp:1.0.0 localhost:8080/myapp:1.0.0
docker tag myapp:1.0.0 localhost:8080/myapp:v1
```

### 3.2 推送镜像

```bash
# 推送单个镜像
docker push localhost:8080/myapp:latest

# 推送所有标签
docker push localhost:8080/myapp --all-tags
```

### 3.3 推送示例输出

```
The push refers to repository [localhost:8080/myapp]
5f70bf18a086: Pushed
a3ed95caeb02: Pushed
latest: digest: sha256:abc123... size: 1234
```

---

## 四、镜像拉取（从仓库获取）

### 4.1 拉取镜像

```bash
# 拉取指定标签
docker pull localhost:8080/myapp:latest

# 拉取指定版本
docker pull localhost:8080/myapp:1.0.0
```

### 4.2 查看已拉取镜像

```bash
docker images | grep localhost:8080
```

---

## 五、镜像管理

### 5.1 通过 Web 界面管理

1. 登录 Web 管理界面 `http://localhost:8080`
2. 进入「镜像管理」页面
3. 可进行以下操作：
   - 查看镜像列表
   - 搜索镜像
   - 查看镜像详情（层信息、大小、创建时间）
   - 删除镜像
   - 获取拉取命令

### 5.2 通过 API 管理

**列出所有镜像：**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/images
```

**搜索镜像：**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/images/search?q=myapp"
```

**获取镜像详情：**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/images/myapp/latest
```

**删除镜像：**
```bash
curl -X DELETE -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/images/myapp/latest
```

### 5.3 列出镜像标签

```bash
# 通过 Docker Registry API
curl http://localhost:8080/v2/myapp/tags/list

# 响应示例
{"name":"myapp","tags":["latest","v1.0.0","v1.1.0"]}
```

---

## 六、镜像备份与导出

### 6.1 导出镜像到文件

```bash
# 从仓库拉取后导出
docker pull localhost:8080/myapp:latest
docker save localhost:8080/myapp:latest -o myapp-latest.tar

# 压缩导出
docker save localhost:8080/myapp:latest | gzip > myapp-latest.tar.gz
```

### 6.2 批量导出

```bash
# 导出多个镜像到单个文件
docker save \
  localhost:8080/myapp:latest \
  localhost:8080/myapp:v1.0.0 \
  -o myapp-all.tar
```

### 6.3 从文件导入镜像

```bash
# 导入镜像
docker load -i myapp-latest.tar

# 从压缩文件导入
gunzip -c myapp-latest.tar.gz | docker load
```

---

## 七、镜像同步

### 7.1 同步到外部仓库

通过 API 将本地镜像同步到 Docker Hub 或其他仓库：

```bash
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "image_name": "myapp",
    "image_tag": "latest",
    "target_registry": "docker.io",
    "target_name": "username/myapp",
    "target_tag": "latest"
  }' \
  http://localhost:8080/api/sync
```

### 7.2 配置同步凭证

```bash
# 保存目标仓库凭证
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "registry": "docker.io",
    "username": "your_username",
    "password": "your_password"
  }' \
  http://localhost:8080/api/credentials
```

### 7.3 查看同步历史

```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/sync/history
```

---

## 八、镜像加速拉取

### 8.1 通过加速器拉取外部镜像

系统内置镜像加速功能，可加速拉取 Docker Hub 等外部镜像：

```bash
# 通过加速器拉取
curl http://localhost:8080/api/accel/pull/library/nginx/manifests/latest
```

### 8.2 查看缓存状态

```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/accel/cache/stats
```

---

## 九、安全功能

### 9.1 镜像签名

系统支持 Sigstore/cosign 镜像签名，确保镜像完整性：

```bash
# 签名镜像（需配置签名密钥）
cosign sign localhost:8080/myapp:latest

# 验证签名
cosign verify localhost:8080/myapp:latest
```

### 9.2 漏洞扫描

推送镜像后，系统自动进行漏洞扫描，可在 Web 界面查看扫描结果。

### 9.3 SBOM 生成

系统自动为推送的镜像生成软件物料清单（SBOM），便于供应链安全审计。

---

## 十、常见问题

### Q1: 推送镜像时提示认证失败

```bash
# 重新登录
docker logout localhost:8080
docker login localhost:8080
```

### Q2: 推送大镜像超时

修改 Docker 客户端超时配置，或分层推送：

```bash
# 检查网络连接
curl -v http://localhost:8080/v2/
```

### Q3: 磁盘空间不足

```bash
# 查看存储使用情况
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/system/info

# 清理未使用的镜像层
curl -X DELETE -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/accel/cache
```

### Q4: 如何迁移镜像数据

1. 停止服务
2. 备份 `data/blobs` 和 `data/meta` 目录
3. 在新环境恢复数据目录
4. 启动服务

---

## 十一、最佳实践

1. **标签规范**：使用语义化版本号（如 `v1.0.0`），避免仅使用 `latest`
2. **定期清理**：配置自动清理策略，删除过期镜像
3. **备份策略**：定期导出重要镜像到外部存储
4. **安全扫描**：关注漏洞扫描结果，及时更新存在安全问题的镜像
5. **访问控制**：使用组织命名空间和 RBAC 控制镜像访问权限

---

## 十二、快速参考

| 操作 | 命令 |
|------|------|
| 登录仓库 | `docker login localhost:8080` |
| 标记镜像 | `docker tag <镜像> localhost:8080/<名称>:<标签>` |
| 推送镜像 | `docker push localhost:8080/<名称>:<标签>` |
| 拉取镜像 | `docker pull localhost:8080/<名称>:<标签>` |
| 导出镜像 | `docker save <镜像> -o <文件>.tar` |
| 导入镜像 | `docker load -i <文件>.tar` |
| 查看标签 | `curl http://localhost:8080/v2/<名称>/tags/list` |

---

**文档版本**: v1.0.0  
**最后更新**: 2026-01-16  
**作者**: CYP | nasDSSCYP@outlook.com
