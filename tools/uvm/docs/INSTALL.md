# 安装指南

## 安装方式

### 方式 1: NPM 安装（推荐）

```bash
# 全局安装
npm install -g @cyp/unified-version-system

# 项目安装
npm install --save-dev @cyp/unified-version-system
```

### 方式 2: 从源码安装

```bash
# 克隆仓库
git clone https://github.com/your-repo/unified-version-system.git

# 进入目录
cd unified-version-system

# 全局链接
npm link
```

### 方式 3: 直接复制

将整个 `unified-version-system` 目录复制到你的项目中。

---

## 验证安装

### 全局安装验证

```bash
# 查看版本
uvm version

# 查看帮助
uvm help
```

### 项目安装验证

```bash
# 使用 npx
npx uvm version

# 或在 package.json 中添加脚本
{
  "scripts": {
    "version:info": "uvm info"
  }
}
```

---

## 初始化项目

在你的项目根目录运行：

```bash
# 查看当前版本信息
uvm info

# 如果没有 VERSION 文件，创建一个
echo "1.0.0" > VERSION

# 验证系统
uvm validate
```

---

## 配置 package.json

在你的 `package.json` 中添加以下脚本：

```json
{
  "scripts": {
    "version:patch": "uvm patch",
    "version:minor": "uvm minor",
    "version:major": "uvm major",
    "version:update": "uvm update",
    "version:info": "uvm info",
    "version:validate": "uvm validate",
    "version:history": "uvm history",
    "version:history-stats": "uvm history stats"
  }
}
```

---

## 快速开始

```bash
# 1. 查看当前版本
uvm info

# 2. 递增补丁版本
uvm patch

# 3. 查看历史统计
uvm history stats
```

---

## 系统要求

- **Node.js**: >= 14.0.0
- **NPM**: >= 6.0.0
- **操作系统**: Windows, macOS, Linux

---

## 目录结构

安装后，系统会在项目根目录创建以下文件：

```
your-project/
├── VERSION                        # 主版本文件
├── .version/                      # 版本数据目录
│   ├── changelog.json             # 版本历史（JSON）
│   └── VERSION_HISTORY.md         # 版本历史（Markdown）
└── package.json                   # 项目配置
```

---

## 卸载

### 全局卸载

```bash
npm uninstall -g @cyp/unified-version-system
```

### 项目卸载

```bash
npm uninstall @cyp/unified-version-system
```

---

## 故障排除

### 问题 1: 命令未找到

**错误：** `uvm: command not found`

**解决方案：**
```bash
# 检查全局安装
npm list -g @cyp/unified-version-system

# 重新安装
npm install -g @cyp/unified-version-system
```

### 问题 2: 权限错误

**错误：** `EACCES: permission denied`

**解决方案：**
```bash
# 使用 sudo (Linux/macOS)
sudo npm install -g @cyp/unified-version-system

# 或配置 npm 前缀
npm config set prefix ~/.npm-global
export PATH=~/.npm-global/bin:$PATH
```

### 问题 3: 版本文件不存在

**错误：** `VERSION 文件不存在`

**解决方案：**
```bash
# 创建 VERSION 文件
echo "1.0.0" > VERSION

# 或使用初始版本
uvm update 1.0.0
```

---

## 获取帮助

- **文档**: [README.md](./README.md)
- **快速开始**: [QUICK_START.md](./QUICK_START.md)
- **命令参考**: [COMMANDS.md](./COMMANDS.md)
- **问题反馈**: [GitHub Issues](https://github.com/your-repo/unified-version-system/issues)

---

**安装完成后，运行 `uvm help` 查看所有可用命令！**
