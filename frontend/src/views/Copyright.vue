<template>
  <div class="page-layout">
    <div class="page-header">
      <h1 class="page-title">版权信息</h1>
      <p class="page-subtitle">查看项目许可证、版本信息和相关链接</p>
    </div>
    
    <!-- 内容区域 -->
    <div class="content-grid">
      <!-- 许可证信息卡片 -->
      <div class="content-card">
        <div class="content-card-header">
          <el-icon><Document /></el-icon>
          <div>
            <h3 class="content-card-title">许可证信息</h3>
            <p class="content-card-description">MIT License 开源许可证</p>
          </div>
        </div>
        
        <div class="license-content">          
          <div class="license-text">
            <p><strong>版权所有 (c) 2024 本项目贡献者</strong></p>
            <br>
            <p>特此免费授予任何获得本软件副本及相关文档文件（以下简称"软件"）的人不受限制地处理软件的权利，包括但不限于使用、复制、修改、合并、发布、分发、再许可和/或销售软件副本的权利，并允许获得软件的人这样做，但须符合以下条件：</p>
            <br>
            <p>上述版权声明和本许可声明应包含在软件的所有副本或实质性部分中。</p>
            <br>
            <p><strong>免责声明：</strong>本软件按"原样"提供，不提供任何明示或暗示的保证，包括但不限于适销性、特定用途适用性和非侵权性的保证。在任何情况下，作者或版权持有人均不对因软件或软件的使用或其他交易而产生的任何索赔、损害或其他责任承担责任，无论是在合同、侵权还是其他方面。</p>
          </div>
        </div>
      </div>
      
      <!-- 原始项目信息卡片 -->
      <div class="content-card">
        <div class="content-card-header">
          <el-icon><Link /></el-icon>
          <div>
            <h3 class="content-card-title">原始项目信息</h3>
            <p class="content-card-description">基于开源项目进行二次开发</p>
          </div>
        </div>
        
        <div class="project-content">
          <el-alert
            title="基于开源项目进行二次开发"
            type="success"
            :closable="false"
            show-icon
            class="mb-lg"
          />
          
          <el-descriptions :column="1" border>
            <el-descriptions-item label="项目名称">
              monica-proxy
            </el-descriptions-item>
            <el-descriptions-item label="原始作者">
              ycvk
            </el-descriptions-item>
            <el-descriptions-item label="项目地址">
              <el-link
                href="https://github.com/ycvk/monica-proxy"
                target="_blank"
                type="primary"
              >
                https://github.com/ycvk/monica-proxy
                <el-icon><Link /></el-icon>
              </el-link>
            </el-descriptions-item>
            <el-descriptions-item label="许可证">
              MIT License
            </el-descriptions-item>
            <el-descriptions-item label="本项目特点">
              <el-tag type="primary" class="mr-sm">界面优化</el-tag>
              <el-tag type="success" class="mr-sm">功能扩展</el-tag>
              <el-tag type="warning">多GUI框架支持</el-tag>
            </el-descriptions-item>
          </el-descriptions>
          
          <div class="project-description">
            <h4>项目说明</h4>
            <p>本项目是基于 <strong>https://github.com/ycvk/monica-proxy</strong> 项目进行的二次开发。</p>
            <p>原始项目作者：<strong>ycvk</strong></p>
            <p>原始项目许可证：<strong>MIT License</strong></p>
            <p>感谢原作者 <strong>ycvk</strong> 的杰出贡献，本项目在原项目基础上进行了功能扩展和界面优化，并提供了多种GUI框架支持（包括原生的Fyne框架和基于Web的Wails框架）。</p>
          </div>
        </div>
      </div>
      
      <!-- 快速链接卡片 -->
      <div class="content-card">
        <div class="content-card-header">
          <el-icon><Share /></el-icon>
          <div>
            <h3 class="content-card-title">快速链接</h3>
            <p class="content-card-description">相关资源和文档链接</p>
          </div>
        </div>
        
        <div class="links-grid">
          <div class="link-item" @click="openOriginalProject">
            <div class="link-icon">
              <el-icon size="32"><Link /></el-icon>
            </div>
            <div class="link-info">
              <h4>原始项目</h4>
              <p>访问原始项目GitHub仓库</p>
            </div>
            <button class="btn btn-primary btn-sm">访问项目</button>
          </div>
          
          <div class="link-item" @click="showLicenseFull">
            <div class="link-icon success">
              <el-icon size="32"><Document /></el-icon>
            </div>
            <div class="link-info">
              <h4>许可证全文</h4>
              <p>查看MIT许可证完整内容</p>
            </div>
            <button class="btn btn-success btn-sm">查看许可证</button>
          </div>
        </div>
      </div>
      
      <!-- 版本信息卡片 -->
      <div class="content-card">
        <div class="content-card-header">
          <el-icon><InfoFilled /></el-icon>
          <div>
            <h3 class="content-card-title">版本信息</h3>
            <p class="content-card-description">应用版本和构建信息</p>
          </div>
        </div>
        
        <div class="version-content">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="应用名称">
              Monica Proxy
            </el-descriptions-item>
            <el-descriptions-item label="版本">
              1.0.0
            </el-descriptions-item>
            <el-descriptions-item label="GUI框架">
              Wails v2 (本版本) / Fyne v2 (原版)
            </el-descriptions-item>
            <el-descriptions-item label="构建时间">
              {{ buildTime }}
            </el-descriptions-item>
            <el-descriptions-item label="Go版本">
              {{ goVersion }}
            </el-descriptions-item>
            <el-descriptions-item label="开发语言">
              Go + Vue.js + Element Plus
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </div>
    </div>
    
    <!-- 许可证全文对话框 -->
    <el-dialog
      v-model="showLicenseDialog"
      title="MIT License 完整内容"
      width="60%"
      top="5vh"
    >
      <div class="license-full-text">
        <pre>
MIT License

Copyright (c) 2024 Monica Proxy Project Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
        </pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const showLicenseDialog = ref(false)

const buildTime = ref(new Date().toLocaleString('zh-CN'))
const goVersion = ref('1.23.0')

function openOriginalProject() {
  window.open('https://github.com/ycvk/monica-proxy', '_blank')
}

function showLicenseFull() {
  showLicenseDialog.value = true
}
</script>

<style scoped>
/* 页面布局 */
.page-layout {
  padding: var(--spacing-lg);
  background: var(--background-page);
  min-height: 100vh;
}

/* 内容网格布局 */
.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
  gap: var(--card-gap);
}

/* 内容卡片样式 */
.content-card {
  background: var(--background-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-light);
  transition: all var(--transition-normal);
  overflow: hidden;
}

.content-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

/* 卡片头部样式 */
.content-card-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-lg);
  background: var(--background-section);
  border-bottom: 1px solid var(--border-light);
}

.content-card-header .el-icon {
  font-size: var(--font-size-xl);
  color: var(--primary-color);
}

.content-card-title {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-bold);
  color: var(--text-primary);
}

.content-card-description {
  margin: var(--spacing-xs) 0 0 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

/* 许可证内容样式 */
.license-content {
  padding: var(--spacing-lg);
}

.license-text {
  line-height: var(--line-height-relaxed);
  color: var(--text-regular);
  font-size: var(--font-size-sm);
  background: var(--background-section);
  padding: var(--spacing-lg);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-light);
}

.license-text p {
  margin-bottom: var(--spacing-sm);
}

.license-text strong {
  color: var(--text-primary);
  font-weight: var(--font-weight-bold);
}

/* 项目内容样式 */
.project-content {
  padding: var(--spacing-lg);
}

.project-description {
  margin-top: var(--spacing-lg);
  padding: var(--spacing-lg);
  background: var(--background-section);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-light);
}

.project-description h4 {
  margin: 0 0 var(--spacing-md) 0;
  color: var(--text-primary);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-bold);
}

.project-description p {
  line-height: var(--line-height-normal);
  margin-bottom: var(--spacing-sm);
  color: var(--text-regular);
}

.project-description strong {
  color: var(--primary-color);
}

/* 链接网格样式 */
.links-grid {
  padding: var(--spacing-lg);
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--spacing-lg);
}

.link-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg);
  background: var(--background-section);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-light);
  cursor: pointer;
  transition: all var(--transition-normal);
}

.link-item:hover {
  background: var(--background-hover);
  transform: translateY(-2px);
  box-shadow: var(--shadow-sm);
}

.link-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: var(--gradient-primary);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.link-icon.success {
  background: var(--gradient-success);
}

.link-info {
  flex: 1;
}

.link-info h4 {
  margin: 0 0 var(--spacing-xs) 0;
  color: var(--text-primary);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-bold);
}

.link-info p {
  margin: 0;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
}

/* 按钮样式 */
.btn-sm {
  padding: var(--spacing-sm) var(--spacing-md);
  font-size: var(--font-size-sm);
  min-width: 80px;
  height: 32px;
}

/* 版本内容样式 */
.version-content {
  padding: var(--spacing-lg);
}

/* 许可证全文对话框样式 */
.license-full-text {
  max-height: 70vh;
  overflow-y: auto;
  padding: var(--spacing-lg);
  background: var(--background-section);
  border-radius: var(--radius-md);
}

.license-full-text pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: 'Courier New', monospace;
  font-size: var(--font-size-sm);
  line-height: var(--line-height-normal);
  color: var(--text-primary);
}

/* 间距工具类 */
.mb-lg {
  margin-bottom: var(--spacing-lg);
}

.mr-sm {
  margin-right: var(--spacing-sm);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-layout {
    padding: var(--spacing-md);
  }
  
  .content-grid {
    grid-template-columns: 1fr;
  }
  
  .content-card-header {
    padding: var(--spacing-md);
  }
  
  .license-content,
  .project-content,
  .links-grid,
  .version-content {
    padding: var(--spacing-md);
  }
  
  .link-item {
    flex-direction: column;
    text-align: center;
    gap: var(--spacing-md);
  }
}

@media (max-width: 480px) {
  .page-layout {
    padding: var(--spacing-sm);
  }
  
  .content-card-header {
    flex-direction: column;
    text-align: center;
    gap: var(--spacing-sm);
  }
  
  .license-text,
  .project-description {
    padding: var(--spacing-md);
  }
}
</style>