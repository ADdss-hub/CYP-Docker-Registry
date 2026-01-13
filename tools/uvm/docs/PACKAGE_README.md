# 📦 统一版本管理系统 - 独立包

> 这是一个完整的、可独立使用的版本管理系统包

## ✨ 特点

- 🚀 **开箱即用** - 无需额外配置
- 📦 **完全独立** - 不依赖外部项目
- 🔧 **易于集成** - 支持多种使用方式
- 📚 **文档完整** - 8个详细文档
- 🎯 **功能强大** - 自动化版本管理

---

## 🎯 这个目录包含什么？

这个 `unified-version-system` 目录是一个**完整的 NPM 包**，包含：

✅ 核心功能模块（4个）
✅ 命令行工具（CLI）
✅ 完整文档（8个）
✅ 使用示例
✅ MIT 许可证

---

## 🚀 快速开始

### 1. 全局安装（推荐）

```bash
# 在此目录下运行
npm link

# 然后在任何项目中使用
uvm help
uvm patch
uvm info
```

### 2. 项目本地使用

```bash
# 复制到你的项目
cp -r unified-version-system /path/to/your/project/

# 在项目中使用
node unified-version-system/bin/uvm.js help
```

### 3. 发布到 NPM

```bash
npm login
npm publish
```

---

## 📝 基本命令

```bash
# 版本更新
uvm patch              # 1.0.0 -> 1.0.1
uvm minor              # 1.0.0 -> 1.1.0
uvm major              # 1.0.0 -> 2.0.0

# 版本信息
uvm info               # 显示详细信息
uvm current            # 显示当前版本
uvm validate           # 验证系统

# 版本历史
uvm history            # 生成历史文档
uvm history stats      # 显示统计
```

---

## 📚 文档导航

| 文档 | 说明 |
|------|------|
| [README.md](./README.md) | 完整的系统说明 |
| [INSTALL.md](./INSTALL.md) | 安装指南 |
| [QUICK_START.md](./QUICK_START.md) | 快速开始 |
| [COMMANDS.md](./COMMANDS.md) | 命令参考 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 架构设计 |
| [FORMAT_SPECIFICATION.md](./FORMAT_SPECIFICATION.md) | 格式规范 |
| [如何使用.md](./如何使用.md) | 使用方式 |
| [使用说明.txt](./使用说明.txt) | 简明说明 |

---

## 🎁 分发方式

### 方式 1: 直接分享目录

```bash
# 压缩整个目录
zip -r unified-version-system.zip unified-version-system/

# 分享 zip 文件
```

### 方式 2: 发布到 NPM

```bash
cd unified-version-system
npm publish
```

### 方式 3: Git 仓库

```bash
# 创建独立仓库
git init
git add .
git commit -m "Initial commit"
git remote add origin <your-repo-url>
git push -u origin main
```

---

## 🔧 系统要求

- Node.js >= 14.0.0
- NPM >= 6.0.0
- Windows / macOS / Linux

---

## 📦 包信息

```json
{
  "name": "@cyp/unified-version-system",
  "version": "1.0.0",
  "description": "统一版本管理系统",
  "author": "CYP",
  "license": "MIT"
}
```

---

## 🎯 使用场景

### 场景 1: 在多个项目中使用

```bash
# 全局安装一次
npm link

# 在项目 A 中使用
cd /path/to/project-a
uvm patch

# 在项目 B 中使用
cd /path/to/project-b
uvm minor
```

### 场景 2: 团队共享

```bash
# 1. 压缩目录
zip -r version-system.zip unified-version-system/

# 2. 分享给团队成员

# 3. 团队成员解压并安装
unzip version-system.zip
cd unified-version-system
npm link
```

### 场景 3: CI/CD 集成

```yaml
# .github/workflows/release.yml
- name: Install version manager
  run: |
    cd unified-version-system
    npm link

- name: Bump version
  run: uvm patch
```

---

## ✅ 验证安装

```bash
# 1. 查看版本
uvm version

# 2. 查看帮助
uvm help

# 3. 测试功能（需要在有 VERSION 文件的项目中）
uvm info
```

---

## 🆘 获取帮助

- 运行 `uvm help` 查看所有命令
- 查看 [README.md](./README.md) 了解详细信息
- 查看 [QUICK_START.md](./QUICK_START.md) 快速上手

---

## 📄 许可证

MIT License - 详见 [LICENSE](./LICENSE) 文件

---

## 👤 作者

**CYP**
- 邮箱: nasDSSCYP@outlook.com

---

**这个目录可以直接使用、分发或发布！** 🎉
