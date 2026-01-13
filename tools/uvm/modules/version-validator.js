#!/usr/bin/env node

/**
 * 版本验证模块
 * 负责验证版本号格式和硬编码检查
 * 
 * @module version-validator
 * @author CYP
 * @version v1.15.7
 */

const fs = require('fs');
const path = require('path');

class VersionValidator {
  constructor(options = {}) {
    this.projectRoot = options.projectRoot || process.cwd();
    this.silent = options.silent || false;
  }

  /**
   * 验证版本号格式
   * @param {string} version - 版本号
   * @returns {Object} 验证结果
   */
  validateFormat(version) {
    const result = {
      valid: false,
      version: version,
      errors: [],
      warnings: []
    };

    // 移除 v 前缀
    const cleanVersion = version.replace(/^v/, '');
    
    // 检查格式
    const versionPattern = /^\d+\.\d+\.\d+$/;
    if (!versionPattern.test(cleanVersion)) {
      result.errors.push('版本号格式错误，应为 x.y.z 格式');
      return result;
    }

    // 解析版本号
    const [major, minor, patch] = cleanVersion.split('.').map(Number);
    
    if (major < 0 || minor < 0 || patch < 0) {
      result.errors.push('版本号不能包含负数');
      return result;
    }

    result.valid = true;
    result.parsed = { major, minor, patch };
    
    return result;
  }

  /**
   * 检查硬编码问题
   * @returns {Object} 检查结果
   */
  checkHardcode() {
    const filesToCheck = [
      'scripts/update-version.js',
      'scripts/version-manager.js',
      'unified-version-system/modules/version-writer.js'
    ];

    const hardcodePatterns = [
      /APP_VERSION\s*=\s*["']v?\d+\.\d+\.\d+["']/g,
      /VERSION_NUMBER\s*=\s*["']v?\d+\.\d+\.\d+["']/g,
      /version:\s*["']v?\d+\.\d+\.\d+["']/g,
      /fullversion:\s*["']v?\d+\.\d+\.\d+["']/g
    ];

    const allowedPatterns = [
      /APP_VERSION\s*=\s*["`]\$\{/,
      /VERSION_NUMBER\s*=\s*["`]\$\{/,
      /version:\s*["`]\$\{/,
      /fullversion:\s*["`]\$\{/
    ];

    const result = {
      valid: true,
      errors: [],
      files: {}
    };

    filesToCheck.forEach(file => {
      const filePath = path.join(this.projectRoot, file);
      
      if (!fs.existsSync(filePath)) {
        return;
      }

      const content = fs.readFileSync(filePath, 'utf8');
      const lines = content.split('\n');
      const fileErrors = [];
      
      lines.forEach((line, index) => {
        if (line.trim().startsWith('//') || line.trim().startsWith('*')) {
          return;
        }
        
        hardcodePatterns.forEach(pattern => {
          const matches = line.match(pattern);
          if (matches) {
            const isAllowed = allowedPatterns.some(allowedPattern => 
              allowedPattern.test(line)
            );
            
            if (!isAllowed) {
              fileErrors.push({
                line: index + 1,
                content: line.trim(),
                message: '发现硬编码版本号'
              });
              result.valid = false;
            }
          }
        });
      });
      
      if (fileErrors.length > 0) {
        result.files[file] = fileErrors;
        result.errors.push(...fileErrors.map(e => `${file}:${e.line} - ${e.message}`));
      }
    });

    return result;
  }

  /**
   * 执行完整验证
   * @param {string} version - 版本号
   * @returns {Object} 验证结果
   */
  validate(version) {
    if (!this.silent) {
      console.log('🔍 开始版本验证...\n');
    }

    const results = {
      valid: true,
      format: null,
      hardcode: null
    };

    // 验证格式
    results.format = this.validateFormat(version);
    if (!results.format.valid) {
      results.valid = false;
      if (!this.silent) {
        console.log('❌ 版本格式验证失败:');
        results.format.errors.forEach(err => console.log(`   ${err}`));
      }
    } else if (!this.silent) {
      console.log('✅ 版本格式验证通过');
    }

    // 检查硬编码
    results.hardcode = this.checkHardcode();
    if (!results.hardcode.valid) {
      results.valid = false;
      if (!this.silent) {
        console.log('\n❌ 硬编码检查失败:');
        Object.keys(results.hardcode.files).forEach(file => {
          console.log(`\n   文件: ${file}`);
          results.hardcode.files[file].forEach(err => {
            console.log(`     第 ${err.line} 行: ${err.content}`);
          });
        });
      }
    } else if (!this.silent) {
      console.log('✅ 硬编码检查通过');
    }

    if (!this.silent) {
      console.log('');
    }

    return results;
  }

  /**
   * 生成验证报告
   * @param {Object} results - 验证结果
   * @returns {string} 报告内容
   */
  generateReport(results) {
    let report = '# 版本验证报告\n\n';
    report += `生成时间: ${new Date().toISOString()}\n\n`;
    
    report += '## 格式验证\n\n';
    if (results.format.valid) {
      report += '✅ 通过\n\n';
      report += `- 版本号: ${results.format.version}\n`;
      report += `- 主版本: ${results.format.parsed.major}\n`;
      report += `- 次版本: ${results.format.parsed.minor}\n`;
      report += `- 补丁版本: ${results.format.parsed.patch}\n`;
    } else {
      report += '❌ 失败\n\n';
      results.format.errors.forEach(err => {
        report += `- ${err}\n`;
      });
    }
    
    report += '\n## 硬编码检查\n\n';
    if (results.hardcode.valid) {
      report += '✅ 通过\n';
    } else {
      report += '❌ 失败\n\n';
      Object.keys(results.hardcode.files).forEach(file => {
        report += `### ${file}\n\n`;
        results.hardcode.files[file].forEach(err => {
          report += `- 第 ${err.line} 行: ${err.message}\n`;
          report += `  \`\`\`\n  ${err.content}\n  \`\`\`\n`;
        });
        report += '\n';
      });
    }
    
    return report;
  }
}

module.exports = VersionValidator;

// CLI 支持
if (require.main === module) {
  const validator = new VersionValidator();
  const version = process.argv[2] || require(path.join(process.cwd(), 'VERSION'));
  
  const results = validator.validate(version);
  
  if (!results.valid) {
    console.log('💡 修复建议:');
    console.log('   1. 确保版本号格式为 x.y.z');
    console.log('   2. 将硬编码改为使用变量 ${version}');
    console.log('   3. 运行 npm run version:validate 重新验证\n');
    process.exit(1);
  }
  
  console.log('✅ 所有验证通过！\n');
  process.exit(0);
}
