#!/usr/bin/env node

/**
 * 版本写入模块
 * 负责将版本号写入各个文件
 * 
 * @module version-writer
 * @author CYP
 * @version v1.15.8
 */

const fs = require('fs');
const path = require('path');

class VersionWriter {
  constructor(options = {}) {
    this.projectRoot = options.projectRoot || process.cwd();
    this.silent = options.silent || false;
  }

  /**
   * 使用正则替换文件中的版本号
   * @param {string} filePath - 文件路径
   * @param {Array} patterns - 替换模式数组 [{search: RegExp, replace: string}]
   * @param {string} displayName - 显示名称
   * @returns {boolean} 是否成功替换
   */
  replaceInFile(filePath, patterns, displayName) {
    const fullPath = path.join(this.projectRoot, filePath);
    
    if (!fs.existsSync(fullPath)) {
      if (!this.silent) {
        console.log(`  ⚠ ${displayName}: 文件不存在`);
      }
      return false;
    }

    try {
      let content = fs.readFileSync(fullPath, 'utf8');
      let changed = false;

      patterns.forEach(pattern => {
        if (pattern.search.test(content)) {
          content = content.replace(pattern.search, pattern.replace);
          changed = true;
        }
        // 重置正则的 lastIndex
        pattern.search.lastIndex = 0;
      });

      if (changed) {
        fs.writeFileSync(fullPath, content);
        if (!this.silent) {
          console.log(`  ✓ ${displayName}`);
        }
        return true;
      } else {
        if (!this.silent) {
          console.log(`  ⏭ ${displayName}: 已是最新或未找到匹配`);
        }
        return false;
      }
    } catch (error) {
      if (!this.silent) {
        console.warn(`  ⚠ ${displayName}: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 写入 VERSION 文件
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeVersionFile(version) {
    const versionFile = path.join(this.projectRoot, 'VERSION');
    const cleanVersion = version.replace(/^v/, '');
    
    fs.writeFileSync(versionFile, cleanVersion + '\n');
    
    if (!this.silent) {
      console.log(`  ✓ VERSION 文件: ${cleanVersion}`);
    }
  }

  /**
   * 写入 package.json 文件
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writePackageJson(version) {
    const cleanVersion = version.replace(/^v/, '');
    const packageFiles = [
      'package.json',
      'frontend/package.json',
      'backend/package.json',
      'packages/app/package.json',
      'packages/admin/package.json',
      'packages/shared/package.json',
      'packages/server/package.json'
    ];

    packageFiles.forEach(file => {
      const filePath = path.join(this.projectRoot, file);
      
      if (fs.existsSync(filePath)) {
        try {
          const packageData = JSON.parse(fs.readFileSync(filePath, 'utf8'));
          packageData.version = cleanVersion;
          fs.writeFileSync(filePath, JSON.stringify(packageData, null, 2) + '\n');
          
          if (!this.silent) {
            console.log(`  ✓ ${file}: ${cleanVersion}`);
          }
        } catch (error) {
          if (!this.silent) {
            console.warn(`  ⚠ ${file}: ${error.message}`);
          }
        }
      }
    });
  }

  /**
   * 写入前端版本文件
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeFrontendVersion(version) {
    const cleanVersion = version.replace(/^v/, '');
    const versionFile = path.join(this.projectRoot, 'frontend/src/utils/version.ts');
    
    if (!fs.existsSync(path.dirname(versionFile))) {
      fs.mkdirSync(path.dirname(versionFile), { recursive: true });
    }

    const buildTime = new Date();
    const buildTimeFormatted = buildTime.toLocaleString('zh-CN', { 
      year: 'numeric', 
      month: '2-digit', 
      day: '2-digit', 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit', 
      hour12: false 
    }).replace(/\//g, '-').replace(/,/g, '');
    
    const content = `/**
 * 应用版本信息
 * 自动生成，请勿手动修改
 * 最后更新: ${buildTime.toISOString()}
 */

export const APP_VERSION = "${cleanVersion}";
export const VERSION_NUMBER = "${cleanVersion}";
export const BUILD_TIME = '${buildTime.toISOString()}';

export const VERSION_INFO = {
  version: "${cleanVersion}",
  versionPlain: '${cleanVersion}',
  projectName: 'CYP-memo',
  buildTime: '${buildTime.toISOString()}',
  buildTimeFormatted: '${buildTimeFormatted}',
  fullversion: "${cleanVersion}",
} as const;

export default VERSION_INFO;
`;

    fs.writeFileSync(versionFile, content);
    
    if (!this.silent) {
      console.log(`  ✓ 前端版本文件: ${cleanVersion}`);
    }
  }

  /**
   * 写入 shared 包版本配置文件
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeSharedVersion(version) {
    const cleanVersion = version.replace(/^v/, '');
    const versionParts = cleanVersion.split('.');
    const major = parseInt(versionParts[0]) || 0;
    const minor = parseInt(versionParts[1]) || 0;
    const patch = parseInt(versionParts[2]) || 0;
    
    const versionFile = path.join(this.projectRoot, 'packages/shared/src/config/version.ts');
    
    if (!fs.existsSync(versionFile)) {
      return;
    }

    const content = `/**
 * CYP-memo 版本信息
 * Copyright (c) 2025 CYP <nasDSSCYP@outlook.com>
 */

export const VERSION = {
  major: ${major},
  minor: ${minor},
  patch: ${patch},
  get full() {
    return \`\${this.major}.\${this.minor}.\${this.patch}\`
  },
  author: 'CYP',
  email: 'nasDSSCYP@outlook.com',
  get copyrightLines() {
    return {
      line1: \`CYP-memo v\${this.full}\`,
      line2: \`作者: \${this.author}\`,
      line3: \`版权所有 © \${new Date().getFullYear()} CYP\`,
      line4: '保留所有权利',
    }
  },
}
`;

    fs.writeFileSync(versionFile, content);
    
    if (!this.silent) {
      console.log(`  ✓ shared 版本配置: ${cleanVersion}`);
    }
  }

  /**
   * 写入 README.md 版本号
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeReadmeVersion(version) {
    const cleanVersion = version.replace(/^v/, '');
    
    this.replaceInFile('README.md', [
      { search: /(\*\*版本\*\*:\s*v?)[\d.]+/g, replace: `$1${cleanVersion}` },
      { search: /(version-)[\d.]+(-blue)/g, replace: `$1${cleanVersion}$2` }
    ], 'README.md');
  }

  /**
   * 写入 web 前端 package.json (CYP-Docker-Registry)
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeWebPackageJson(version) {
    const cleanVersion = version.replace(/^v/, '');
    const webPackageFile = path.join(this.projectRoot, 'web/package.json');
    
    if (fs.existsSync(webPackageFile)) {
      try {
        const packageData = JSON.parse(fs.readFileSync(webPackageFile, 'utf8'));
        packageData.version = cleanVersion;
        fs.writeFileSync(webPackageFile, JSON.stringify(packageData, null, 2) + '\n');
        
        if (!this.silent) {
          console.log(`  ✓ web/package.json: ${cleanVersion}`);
        }
      } catch (error) {
        if (!this.silent) {
          console.warn(`  ⚠ web/package.json: ${error.message}`);
        }
      }
    }
  }

  /**
   * 写入 Go 服务版本号 (CYP-Docker-Registry)
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeGoServiceVersion(version) {
    const cleanVersion = version.replace(/^v/, '');
    
    this.replaceInFile('internal/service/system_service.go', [
      { search: /(Version:\s*")[\d.]+(")/g, replace: `$1${cleanVersion}$2` }
    ], 'internal/service/system_service.go');
  }

  /**
   * 写入 Dockerfile 版本号 (CYP-Docker-Registry)
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeDockerfileVersion(version) {
    const cleanVersion = version.replace(/^v/, '');
    
    this.replaceInFile('Dockerfile', [
      { search: /(# Version: v)[\d.]+/g, replace: `$1${cleanVersion}` },
      { search: /(LABEL version=")[\d.]+(")/g, replace: `$1${cleanVersion}$2` }
    ], 'Dockerfile');
  }

  /**
   * 写入 Shell 脚本版本号 (CYP-Docker-Registry)
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeShellScriptsVersion(version) {
    const cleanVersion = version.replace(/^v/, '');
    
    const shellScripts = [
      {
        file: 'scripts/entrypoint.sh',
        patterns: [
          { search: /(# Version: v)[\d.]+/g, replace: `$1${cleanVersion}` },
          { search: /(CYP-Docker-Registry v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: 'scripts/install.sh',
        patterns: [
          { search: /(# Version: v)[\d.]+/g, replace: `$1${cleanVersion}` },
          { search: /(VERSION=")[\d.]+(")/g, replace: `$1${cleanVersion}$2` },
          { search: /(智能安装脚本 v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: 'scripts/quick-start.sh',
        patterns: [
          { search: /(# Version: v)[\d.]+/g, replace: `$1${cleanVersion}` },
          { search: /(快速启动脚本 v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: 'scripts/unlock.sh',
        patterns: [
          { search: /(# Version: v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: 'scripts/detect-env.sh',
        patterns: [
          { search: /(# Version: v)[\d.]+/g, replace: `$1${cleanVersion}` },
          { search: /(环境检测工具 v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      }
    ];

    shellScripts.forEach(script => {
      this.replaceInFile(script.file, script.patterns, script.file);
    });
  }

  /**
   * 写入项目文档版本号 (CYP-Docker-Registry)
   * @param {string} version - 版本号（不含 v 前缀）
   */
  writeProjectDocsVersion(version) {
    const cleanVersion = version.replace(/^v/, '');
    
    const docFiles = [
      {
        file: 'PROJECT_STATUS.md',
        patterns: [
          { search: /(\*\*设计文档版本\*\*: v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: '宣传文件.md',
        patterns: [
          { search: /(CYP-Docker Registry v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: '设计文档.md',
        patterns: [
          { search: /(\*\*版本\*\*: v)[\d.]+/g, replace: `$1${cleanVersion}` },
          { search: /(version: "v)[\d.]+(")/g, replace: `$1${cleanVersion}$2` },
          { search: /(\*\*文档版本\*\*: v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: 'docs/SECURITY.md',
        patterns: [
          { search: /(\*\*版本\*\*: v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      },
      {
        file: 'docs/DEPLOY.md',
        patterns: [
          { search: /(\*\*版本\*\*: v)[\d.]+/g, replace: `$1${cleanVersion}` }
        ]
      }
    ];

    docFiles.forEach(doc => {
      this.replaceInFile(doc.file, doc.patterns, doc.file);
    });
  }

  /**
   * 写入所有文件
   * @param {string} version - 版本号
   */
  writeAll(version) {
    if (!this.silent) {
      console.log('📝 写入版本号到文件...\n');
    }

    // 核心版本文件
    this.writeVersionFile(version);
    this.writePackageJson(version);
    this.writeFrontendVersion(version);
    this.writeSharedVersion(version);
    this.writeReadmeVersion(version);

    // CYP-Docker-Registry 项目特有文件
    this.writeWebPackageJson(version);
    this.writeGoServiceVersion(version);
    this.writeDockerfileVersion(version);
    this.writeShellScriptsVersion(version);
    this.writeProjectDocsVersion(version);

    if (!this.silent) {
      console.log('');
    }
  }
}

module.exports = VersionWriter;

// CLI 支持
if (require.main === module) {
  const version = process.argv[2];
  
  if (!version) {
    console.error('❌ 请提供版本号');
    console.log('用法: node version-writer.js <version>');
    process.exit(1);
  }

  const writer = new VersionWriter();
  writer.writeAll(version);
  
  console.log('✅ 版本号写入完成！\n');
}
