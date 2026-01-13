#!/usr/bin/env node

/**
 * 版本递增模块
 * 负责版本号的递增计算
 * 
 * @module version-incrementer
 * @author CYP
 * @version v1.15.7
 */

class VersionIncrementer {
  constructor(options = {}) {
    this.silent = options.silent || false;
  }

  /**
   * 解析版本号
   * @param {string} version - 版本号
   * @returns {Object} 解析结果
   */
  parse(version) {
    const cleanVersion = version.replace(/^v/, '');
    const parts = cleanVersion.split('.');
    
    if (parts.length !== 3) {
      throw new Error(`无效的版本号格式: ${version}`);
    }

    return {
      major: parseInt(parts[0], 10),
      minor: parseInt(parts[1], 10),
      patch: parseInt(parts[2], 10)
    };
  }

  /**
   * 递增主版本号
   * @param {string} version - 当前版本号
   * @returns {string} 新版本号
   */
  incrementMajor(version) {
    const parsed = this.parse(version);
    return `${parsed.major + 1}.0.0`;
  }

  /**
   * 递增次版本号
   * @param {string} version - 当前版本号
   * @returns {string} 新版本号
   */
  incrementMinor(version) {
    const parsed = this.parse(version);
    return `${parsed.major}.${parsed.minor + 1}.0`;
  }

  /**
   * 递增补丁版本号
   * @param {string} version - 当前版本号
   * @returns {string} 新版本号
   */
  incrementPatch(version) {
    const parsed = this.parse(version);
    return `${parsed.major}.${parsed.minor}.${parsed.patch + 1}`;
  }

  /**
   * 根据类型递增版本号
   * @param {string} version - 当前版本号
   * @param {string} type - 递增类型 (major/minor/patch)
   * @returns {string} 新版本号
   */
  increment(version, type = 'patch') {
    if (!this.silent) {
      console.log(`📈 递增 ${type} 版本号...\n`);
    }

    let newVersion;
    
    switch (type.toLowerCase()) {
      case 'major':
        newVersion = this.incrementMajor(version);
        break;
      case 'minor':
        newVersion = this.incrementMinor(version);
        break;
      case 'patch':
        newVersion = this.incrementPatch(version);
        break;
      default:
        throw new Error(`不支持的递增类型: ${type}`);
    }

    if (!this.silent) {
      console.log(`  ${version} → ${newVersion}\n`);
    }

    return newVersion;
  }

  /**
   * 比较两个版本号
   * @param {string} version1 - 版本号1
   * @param {string} version2 - 版本号2
   * @returns {number} -1: v1 < v2, 0: v1 = v2, 1: v1 > v2
   */
  compare(version1, version2) {
    const v1 = this.parse(version1);
    const v2 = this.parse(version2);

    if (v1.major !== v2.major) {
      return v1.major > v2.major ? 1 : -1;
    }
    if (v1.minor !== v2.minor) {
      return v1.minor > v2.minor ? 1 : -1;
    }
    if (v1.patch !== v2.patch) {
      return v1.patch > v2.patch ? 1 : -1;
    }

    return 0;
  }

  /**
   * 获取版本号建议
   * @param {string} currentVersion - 当前版本号
   * @returns {Object} 版本号建议
   */
  getSuggestions(currentVersion) {
    return {
      current: currentVersion,
      major: this.incrementMajor(currentVersion),
      minor: this.incrementMinor(currentVersion),
      patch: this.incrementPatch(currentVersion)
    };
  }
}

module.exports = VersionIncrementer;

// CLI 支持
if (require.main === module) {
  const currentVersion = process.argv[2];
  const type = process.argv[3] || 'patch';

  if (!currentVersion) {
    console.error('❌ 请提供当前版本号');
    console.log('用法: node version-incrementer.js <current-version> [major|minor|patch]');
    process.exit(1);
  }

  const incrementer = new VersionIncrementer();
  
  try {
    const newVersion = incrementer.increment(currentVersion, type);
    console.log(`✅ 新版本号: ${newVersion}\n`);
  } catch (error) {
    console.error(`❌ 错误: ${error.message}\n`);
    process.exit(1);
  }
}
