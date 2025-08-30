<template>
  <div class="copyright">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>版权信息</span>
        </div>
      </template>
      
      <div class="content">
        <!-- 版权标题 -->
        <div class="section">
          <h2><el-icon><Document /></el-icon> 许可证信息</h2>
          <el-alert
            title="MIT License"
            type="info"
            :closable="false"
            show-icon
            style="margin-bottom: 20px;"
          />
          
          <el-card class="license-card">
            <div class="license-text">
              <p><strong>版权所有 (c) 2024 本项目贡献者</strong></p>
              <br>
              <p>特此免费授予任何获得本软件副本及相关文档文件（以下简称"软件"）的人不受限制地处理软件的权利，包括但不限于使用、复制、修改、合并、发布、分发、再许可和/或销售软件副本的权利，并允许获得软件的人这样做，但须符合以下条件：</p>
              <br>
              <p>上述版权声明和本许可声明应包含在软件的所有副本或实质性部分中。</p>
              <br>
              <p><strong>免责声明：</strong>本软件按"原样"提供，不提供任何明示或暗示的保证，包括但不限于适销性、特定用途适用性和非侵权性的保证。在任何情况下，作者或版权持有人均不对因软件或软件的使用或其他交易而产生的任何索赔、损害或其他责任承担责任，无论是在合同、侵权还是其他方面。</p>
            </div>
          </el-card>
        </div>
        
        <!-- 原始项目信息 -->
        <div class="section">
          <h2><el-icon><Link /></el-icon> 原始项目信息</h2>
          <el-alert
            title="基于开源项目进行二次开发"
            type="success"
            :closable="false"
            show-icon
            style="margin-bottom: 20px;"
          />
          
          <el-card class="original-project-card">
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
                <el-tag type="primary" style="margin-right: 5px;">界面优化</el-tag>
                <el-tag type="success" style="margin-right: 5px;">功能扩展</el-tag>
                <el-tag type="warning">多GUI框架支持</el-tag>
              </el-descriptions-item>
            </el-descriptions>
            
            <div class="project-description">
              <h3>项目说明</h3>
              <p>本项目是基于 <strong>https://github.com/ycvk/monica-proxy</strong> 项目进行的二次开发。</p>
              <p>原始项目作者：<strong>ycvk</strong></p>
              <p>原始项目许可证：<strong>MIT License</strong></p>
              <p>感谢原作者 <strong>ycvk</strong> 的杰出贡献，本项目在原项目基础上进行了功能扩展和界面优化，并提供了多种GUI框架支持（包括原生的Fyne框架和基于Web的Wails框架）。</p>
            </div>
          </el-card>
        </div>
        
        <!-- 快速链接 -->
        <div class="section">
          <h2><el-icon><Share /></el-icon> 快速链接</h2>
          <div class="quick-links">
            <el-row :gutter="20">
              <el-col :span="12">
                <el-card class="link-card" shadow="hover">
                  <div class="link-content">
                    <el-icon size="40" color="#409EFF"><Link /></el-icon>
                    <div class="link-info">
                      <h4>原始项目</h4>
                      <p>访问原始项目GitHub仓库</p>
                      <el-button
                        type="primary"
                        size="small"
                        @click="openOriginalProject"
                      >
                        访问项目
                      </el-button>
                    </div>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="12">
                <el-card class="link-card" shadow="hover">
                  <div class="link-content">
                    <el-icon size="40" color="#67C23A"><Document /></el-icon>
                    <div class="link-info">
                      <h4>许可证全文</h4>
                      <p>查看MIT许可证完整内容</p>
                      <el-button
                        type="success"
                        size="small"
                        @click="showLicenseFull"
                      >
                        查看许可证
                      </el-button>
                    </div>
                  </div>
                </el-card>
              </el-col>
            </el-row>
          </div>
        </div>
        
        <!-- 版本信息 -->
        <div class="section">
          <h2><el-icon><InfoFilled /></el-icon> 版本信息</h2>
          <el-card>
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
          </el-card>
        </div>
      </div>
    </el-card>
    
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
.copyright {
  max-width: 1000px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: bold;
  font-size: 18px;
}

.content {
  padding: 20px 0;
}

.section {
  margin-bottom: 40px;
}

.section h2 {
  color: #409EFF;
  margin-bottom: 15px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.license-card {
  background-color: #f8f9fa;
}

.license-text {
  line-height: 1.8;
  color: #333;
}

.license-text p {
  margin-bottom: 10px;
}

.original-project-card {
  background-color: #f0f9ff;
}

.project-description {
  margin-top: 20px;
  padding: 15px;
  background-color: #e6f3ff;
  border-radius: 4px;
}

.project-description h3 {
  color: #409EFF;
  margin-bottom: 10px;
}

.project-description p {
  line-height: 1.6;
  margin-bottom: 8px;
}

.quick-links {
  margin-top: 20px;
}

.link-card {
  cursor: pointer;
  transition: all 0.3s ease;
}

.link-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.1);
}

.link-content {
  display: flex;
  align-items: center;
  gap: 15px;
}

.link-info {
  flex: 1;
}

.link-info h4 {
  margin: 0 0 5px 0;
  color: #303133;
}

.link-info p {
  margin: 0 0 10px 0;
  color: #909399;
  font-size: 14px;
}

.license-full-text {
  max-height: 70vh;
  overflow-y: auto;
  padding: 20px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.license-full-text pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.5;
}
</style>