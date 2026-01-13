#!/usr/bin/env node

/**
 * 版本写入模块
 * 负责将版本号写入各个文件
 * 
 * @module version-writer
 * @author CYP
 * @version v1.15.7
 */

const fs = require('fs');
const path = require('path');

class VersionWriter {
  constructor(options = {}) {
    this.projectRoot = options.projectRoot || process.cwd();
    this.silent = options.silent || false;
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
      // 兼容旧项目结构
      'frontend/package.json',
      'backend/package.json',
      // 适配 monorepo 结构 (packages/)
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
    
    // ⚠️⚠️⚠️ 严重警告：请勿在此处硬编码版本号！⚠️⚠️⚠️
    // ⚠️ 必须使用变量 ${cleanVersion}，不要写死版本号
    const content = `/**
 * 应用版本信息
 * 自动生成，请勿手动修改
 * 最后更新: ${buildTime.toISOString()}
 */

export const APP_VERSION = "${cleanVersion}";
export const VERSION_NUMBER = "${cleanVersion}";
export const BUILD_TIME = '${buildTime.toISOString()}';

// 版本信息对象
export const VERSION_INFO = {
  version: "${cleanVersion}",
  versionPlain: '${cleanVersion}',
  projectName: 'CYP-memo',
  buildTime: '${buildTime.toISOString()}',
  buildTimeFormatted: '${buildTimeFormatted}',
  fullversion: "${cleanVersion}",
} as const;

// 默认导出
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
    
    // 检查文件是否存在
    if (!fs.existsSync(versionFile)) {
      if (!this.silent) {
        console.log(`  ⚠ shared 版本文件不存在: ${versionFile}`);
      }
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
  /** 分行展示版权信息（优化版） */
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
    const readmeFile = path.join(this.projectRoot, 'README.md');
    
    if (!fs.existsSync(readmeFile)) {
      if (!this.silent) {
        console.log(`  ⚠ README.md 不存在`);
      }
      return;
    }

    try {
      let content = fs.readFileSync(readmeFile, 'utf8');
      
      // 匹配 **版本**: x.x.x 格式
      const versionPattern = /(\*\*版本\*\*:\s*)[\d.]+/;
      if (versionPattern.test(content)) {
        content = content.replace(versionPattern, `$1${cleanVersion}`);
        fs.writeFileSync(readmeFile, content);
        
        if (!this.silent) {
          console.log(`  ✓ README.md: ${cleanVersion}`);
        }
      } else {
        if (!this.silent) {
          console.log(`  ⚠ README.md: 未找到版本号标记`);
        }
      }
    } catch (error) {
      if (!this.silent) {
        console.warn(`  ⚠ README.md: ${error.message}`);
      }
    }
  }

  /**
   * 写入所有文件
   * @param {string} version - 版本号
   */
  writeAll(version) {
    if (!this.silent) {
      console.log('📝 写入版本号到文件...\n');
    }

    this.writeVersionFile(version);
    this.writePackageJson(version);
    this.writeFrontendVersion(version);
    this.writeSharedVersion(version);
    this.writeReadmeVersion(version);

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
