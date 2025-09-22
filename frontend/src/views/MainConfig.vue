<template>
  <div class="page-layout">
    <div class="page-header">
      <h1 class="page-title">主要配置</h1>
      <p class="page-subtitle">配置Monica连接、代理设置、安全选项等核心参数</p>
    </div>
    
    <!-- 状态卡片区域 -->
    <div class="status-row">
      <div class="status-card">
        <div class="status-icon" :class="appStore.isServiceRunning ? 'success' : 'info'">
          <el-icon><VideoPlay v-if="appStore.isServiceRunning" /><VideoPause v-else /></el-icon>
        </div>
        <h3 class="status-title">服务状态</h3>
        <p class="status-description">
          {{ appStore.isServiceRunning ? '服务正在运行' : '服务已停止' }}
        </p>
      </div>
      
      <div class="status-card">
        <div class="status-icon info">
          <el-icon><Connection /></el-icon>
        </div>
        <h3 class="status-title">API状态</h3>
        <p class="status-description">
          {{ appStore.serviceStatus.message || '等待服务启动' }}
        </p>
      </div>
      
      <div class="status-card">
        <div class="status-icon" :class="form.monica.cookie ? 'success' : 'warning'">
          <el-icon><Document /></el-icon>
        </div>
        <h3 class="status-title">配置状态</h3>
        <p class="status-description">
          {{ form.monica.cookie ? '配置已完整' : '需要配置Monica Cookie' }}
        </p>
      </div>
    </div>
    
    <!-- 服务控制区域 -->
    <div class="service-control-section">
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><VideoPlay /></el-icon>
          <div>
            <h3 class="config-card-title">服务控制</h3>
            <p class="config-card-description">启动、停止服务，测试配置和查询额度</p>
          </div>
        </div>
        
        <div class="service-controls">
          <div class="button-group">
            <button class="btn btn-success" @click="startService_btn" :disabled="appStore.isServiceRunning || loading">
              <el-icon><VideoPlay /></el-icon>
              启动服务
            </button>
            
            <button class="btn btn-danger" @click="stopService_btn" :disabled="!appStore.isServiceRunning || loading">
              <el-icon><VideoPause /></el-icon>
              停止服务
            </button>
            
            <button class="btn btn-primary" @click="testConfig" :disabled="!appStore.isServiceRunning || loading">
              <el-icon><Connection /></el-icon>
              测试 API 配置
            </button>
            
            <button class="btn btn-info" @click="getQuota" :disabled="loading">
              <el-icon><Coin /></el-icon>
              查询 Monica 额度
            </button>
          </div>
          
          <!-- 状态显示 -->
          <div class="status-info">
            <el-alert
              :title="appStore.serviceStatus.message"
              :type="appStore.isServiceRunning ? 'success' : 'info'"
              :closable="false"
              show-icon
            />
            
            <div v-if="appStore.serviceStatus.address" class="api-info">
              <p><strong>base_url:</strong> {{ appStore.serviceStatus.address }}</p>
              <p><strong>API Key:</strong> {{ appStore.serviceStatus.apiKey || '未设置' }}</p>
            </div>
            
            <div v-if="quotaInfo.geniusBot !== undefined" class="quota-info">
              <el-alert
                :title="`额度信息: Genius Bot: ${quotaInfo.geniusBot}, Credits: ${quotaInfo.credits}`"
                type="success"
                :closable="false"
                show-icon
              />
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <!-- API端点信息 -->
    <div class="api-endpoints-section">
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><Link /></el-icon>
          <div>
            <h3 class="config-card-title">API端点信息</h3>
            <p class="config-card-description">兼容ChatGPT的API接口</p>
          </div>
        </div>
        
        <div class="api-info">
          <ul>
            <li><strong>POST</strong> /v1/chat/completions - 聊天对话（兼容ChatGPT）</li>
            <li><strong>GET</strong> /v1/models - 获取模型列表</li>
            <li><strong>POST</strong> /v1/images/generations - 图片生成（兼容DALL-E）</li>
          </ul>
        </div>
      </div>
    </div>
    
    <!-- 配置表单区域 -->
    <div class="config-grid">
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><Setting /></el-icon>
          <div>
            <h3 class="config-card-title">Monica配置</h3>
            <p class="config-card-description">配置Monica服务的连接参数</p>
          </div>
        </div>
        
        <el-form :model="form" label-width="120px" size="large" class="form-large">
        <el-form-item label="Cookie*" required>
          <el-input
            v-model="form.monica.cookie"
            type="textarea"
            :rows="4"
            placeholder="请输入Monica登录后的Cookie"
          />
        </el-form-item>
        
        <el-form-item label="Bot UID">
          <el-input
            v-model="form.monica.botUID"
            placeholder="自定义Bot的UID（启用Custom Bot模式时必需）"
            :disabled="!form.monica.enableCustomBotMode"
          />
        </el-form-item>
        
        <el-form-item label="自定义Bot模式">
          <el-switch v-model="form.monica.enableCustomBotMode" />
        </el-form-item>
      </el-form>
      </div>
      
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><Shield /></el-icon>
          <div>
            <h3 class="config-card-title">安全配置</h3>
            <p class="config-card-description">API访问令牌、限流、超时等安全设置</p>
          </div>
        </div>
        
        <el-form :model="form" label-width="120px" size="large" class="form-large">
        <el-form-item label="API Key*" required>
          <el-input
            v-model="form.security.bearerToken"
            placeholder="请输入API访问令牌"
            show-password
          />
        </el-form-item>
        
        <el-form-item label="跳过TLS验证">
          <el-switch v-model="form.security.tlsSkipVerify" />
        </el-form-item>
        
        <el-form-item label="启用限流">
          <el-switch v-model="form.security.rateLimitEnabled" />
        </el-form-item>
        
        <el-form-item label="限流RPS" v-if="form.security.rateLimitEnabled">
          <el-input-number
            v-model="form.security.rateLimitRPS"
            :min="1"
            :max="1000"
          />
        </el-form-item>
        
        <el-form-item label="请求超时(秒)">
          <el-input-number
            v-model="form.security.requestTimeout"
            :min="1"
            :max="300"
          />
        </el-form-item>
      </el-form>
      </div>
      
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><Share /></el-icon>
          <div>
            <h3 class="config-card-title">代理配置</h3>
            <p class="config-card-description">网络代理设置（详细配置在服务器配置页面）</p>
          </div>
        </div>
        
        <el-form :model="form" label-width="120px" size="large" class="form-large">
        <el-form-item label="启用代理">
          <el-switch v-model="form.proxy.enabled" />
          <el-alert
            v-if="form.proxy.enabled && (!form.proxy.httpProxy && !form.proxy.httpsProxy)"
            title="代理已启用但未配置具体地址，请到'服务器配置'页面设置代理地址"
            type="warning"
            :closable="false"
            show-icon
            class="mt-sm"
          />
          <el-alert
            v-if="form.proxy.enabled && (form.proxy.httpProxy || form.proxy.httpsProxy)"
            :title="`代理已启用: ${form.proxy.httpProxy || form.proxy.httpsProxy}`"
            type="success"
            :closable="false"
            show-icon
            class="mt-sm"
          />
        </el-form-item>
      </el-form>
      </div>
    </div>
    
    <!-- 保存配置按钮 -->
    <div class="save-section">
      <button class="btn btn-primary" @click="saveConfig" :loading="loading">
        <el-icon><Check /></el-icon>
        保存配置
      </button>
    </div>
    
    <!-- 测试结果对话框 -->
    <el-dialog
      v-model="showTestResults"
      title="API配置测试结果"
      width="80%"
      top="5vh"
    >
      <div class="test-results">
        <el-collapse v-model="activeCollapse">
          <el-collapse-item
            v-for="(result, index) in testResults"
            :key="index"
            :name="index.toString()"
          >
            <template #title>
              <span :class="getResultClass(result)">
                {{ result.endpoint }} - {{ getResultText(result) }}
              </span>
            </template>
            <div class="test-details">
              <p><strong>请求URL:</strong></p>
              <pre>{{ result.url }}</pre>
              
              <p v-if="result.requestData"><strong>请求数据:</strong></p>
              <pre v-if="result.requestData">{{ result.requestData }}</pre>
              
              <p v-if="result.responseData"><strong>响应数据:</strong></p>
              <pre v-if="result.responseData">{{ result.responseData }}</pre>
              
              <p v-if="result.error"><strong>错误信息:</strong></p>
              <pre v-if="result.error" class="error">{{ result.error }}</pre>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAppStore } from '@/stores/app'
import { VideoPlay, VideoPause, Connection, Coin, Check, Link } from '@element-plus/icons-vue'
import {GetConfig,UpdateConfig,StartService,StopService,TestConfig,GetServiceStatus,GetQuota} from '../../wailsjs/wailsjs/go/main/WailsApp.js'
const appStore = useAppStore()

const form = reactive({
  monica: {
    cookie: '',
    botUID: '',
    enableCustomBotMode: false
  },
  proxy: {
    enabled: false,
    httpProxy: '',
    httpsProxy: '',
    noProxy: ''
  },
  security: {
    bearerToken: '',
    tlsSkipVerify: false,
    rateLimitEnabled: false,
    rateLimitRPS: 10,
    requestTimeout: 30
  }
})

const loading = ref(false)
const showTestResults = ref(false)
const testResults = ref([])
const quotaInfo = ref({})
const activeCollapse = ref([])

onMounted(async () => {
  await loadConfig()
})

async function loadConfig() {
  try {
    const config = await GetConfig()
    if (config.monica) {
      Object.assign(form.monica, config.monica)
    }
    if (config.proxy) {
      Object.assign(form.proxy, config.proxy)
      // 根据是否有代理配置来设置启用状态
      form.proxy.enabled = !!(config.proxy.httpProxy || config.proxy.httpsProxy)
    }
    if (config.security) {
      Object.assign(form.security, config.security)
    }
  } catch (error) {
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('加载配置失败: ' + errorMsg)
  }
}

async function saveConfig() {
  loading.value = true
  try {
    await UpdateConfig({
      monica: form.monica,
      proxy: form.proxy,
      security: form.security
    })
    ElMessage.success('配置保存成功')
  } catch (error) {
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('配置保存失败: ' + errorMsg)
  } finally {
    loading.value = false
  }
}

async function startService_btn() {
  loading.value = true
  
  const loadingMessage = ElMessage({
    message: '正在启动服务，请稍候...',
    type: 'info',
    duration: 0,
    showClose: false
  })
  
  try {
    await saveConfig()
    await StartService()
    loadingMessage.close()
    ElMessage.success('服务启动成功')
    await getServiceStatus1()
  } catch (error) {
    loadingMessage.close()
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('服务启动失败: ' + errorMsg)
  } finally {
    loading.value = false
  }
}

async function stopService_btn() {
  loading.value = true
  
  const loadingMessage = ElMessage({
    message: '正在停止服务，请稍候...',
    type: 'info',
    duration: 0,
    showClose: false
  })
  
  try {
    await StopService()
    loadingMessage.close()
    ElMessage.success('服务停止成功')
    await getServiceStatus1()
  } catch (error) {
    loadingMessage.close()
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('服务停止失败: ' + errorMsg)
  } finally {
    loading.value = false
  }
}

async function getServiceStatus1() {
  try {
    const status = await GetServiceStatus()
    appStore.serviceStatus = status
  } catch (error) {
    console.error('获取服务状态失败:', error)
  }
}

async function testConfig() {
  loading.value = true
  
  // 添加测试开始的用户反馈
  const loadingMessage = ElMessage({
    message: '正在测试 API 配置，请稍候...',
    type: 'info',
    duration: 0, // 不自动关闭
    showClose: false
  })
  
  try {
    await saveConfig()
    const results = await TestConfig()
    testResults.value = results
    showTestResults.value = true
    // 默认全部折叠，不展开任何测试结果
    activeCollapse.value = []
    
    // 关闭加载消息并显示成功消息
    loadingMessage.close()
    ElMessage.success('API 配置测试完成，请查看详细结果')
    
  } catch (error) {
    // 关闭加载消息
    loadingMessage.close()
    
    // 改善错误处理，防止undefined
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('配置测试失败: ' + errorMsg)
    
    // 如果是必填项错误，给出更友好的提示
    if (errorMsg.includes('请填写')) {
      ElMessageBox.alert(
        errorMsg + '\n\n请确保以下必填项已正确填写：\n• Monica Cookie\n• API Key\n• Bot UID（如果启用自定义Bot模式）',
        '配置检查',
        {
          type: 'warning',
          confirmButtonText: '我知道了'
        }
      )
    }
    
  } finally {
    loading.value = false
  }
}

async function getQuota() {
  loading.value = true
  
  const loadingMessage = ElMessage({
    message: '正在查询 Monica 额度，请稍候...',
    type: 'info',
    duration: 0,
    showClose: false
  })
  
  try {
    const quota = await GetQuota()
    loadingMessage.close()
    
    if (quota.error) {
      const errorMsg = quota.error || '未知错误'
      ElMessage.error('获取额度失败: ' + errorMsg)
    } else {
      quotaInfo.value = quota
      ElMessage.success('额度查询成功')
    }
  } catch (error) {
    loadingMessage.close()
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('额度查询失败: ' + errorMsg)
  } finally {
    loading.value = false
  }
}

function getResultClass(result) {
  if (result.error) return 'result-error'
  if (result.statusCode >= 200 && result.statusCode < 300) return 'result-success'
  return 'result-error'
}

function getResultText(result) {
  if (result.error) return `失败: ${result.error}`
  if (result.statusCode >= 200 && result.statusCode < 300) return `成功 (HTTP ${result.statusCode})`
  return `错误 (HTTP ${result.statusCode})`
}
</script>

<style scoped>
/* 页面布局增强 */
.page-layout {
  padding: var(--spacing-lg);
  background: var(--background-page);
  min-height: 100vh;
}

/* 状态卡片样式增强 */
.status-row {
  margin-bottom: var(--spacing-xl);
}

.status-card {
  transition: transform var(--transition-normal);
}

.status-card:hover {
  transform: translateY(-4px);
}

/* 服务控制区域样式 */
.service-control-section {
  margin-bottom: var(--spacing-xl);
}

.service-controls {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.service-controls .button-group {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-md);
  justify-content: center;
  padding: var(--spacing-md) 0;
}

.service-controls .btn {
  padding: var(--spacing-md) var(--spacing-lg);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-medium);
  border-radius: var(--radius-md);
  border: none;
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  min-width: 120px;
  justify-content: center;
}

.service-controls .btn-success {
  background: var(--gradient-success);
  color: white;
}

.service-controls .btn-success:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(103, 194, 58, 0.4);
}

.service-controls .btn-danger {
  background: var(--gradient-error);
  color: white;
}

.service-controls .btn-danger:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(245, 108, 108, 0.4);
}

.service-controls .btn-primary {
  background: var(--gradient-primary);
  color: white;
}

.service-controls .btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.4);
}

.service-controls .btn-info {
  background: var(--gradient-info);
  color: white;
}

.service-controls .btn-info:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(144, 147, 153, 0.4);
}

.service-controls .btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none !important;
}

/* API端点信息区域样式 */
.api-endpoints-section {
  margin-bottom: var(--spacing-xl);
}

/* 配置网格布局 */
.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: var(--card-gap);
  margin-bottom: var(--spacing-xl);
}

/* API端点信息样式 */
.api-info {
  padding: var(--spacing-md);
  background: var(--background-section);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-light);
}

.api-info ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.api-info li {
  padding: var(--spacing-sm) 0;
  border-bottom: 1px solid var(--border-light);
  color: var(--text-regular);
  font-size: var(--font-size-md);
  line-height: var(--line-height-normal);
}

.api-info li:last-child {
  border-bottom: none;
}

.api-info li strong {
  color: var(--primary-color);
  font-weight: var(--font-weight-bold);
  margin-right: var(--spacing-sm);
}

/* 测试结果对话框样式 */
.test-results {
  max-height: 70vh;
  overflow-y: auto;
}

.test-details {
  padding: var(--spacing-md);
}

.test-details pre {
  background: var(--background-section);
  padding: var(--spacing-md);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-light);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-primary);
}

.test-details .error {
  background: rgba(245, 108, 108, 0.1);
  color: var(--error-color);
  border-color: var(--error-color);
}

.result-success {
  color: var(--success-color);
  font-weight: var(--font-weight-bold);
}

.result-error {
  color: var(--error-color);
  font-weight: var(--font-weight-bold);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-layout {
    padding: var(--spacing-md);
  }
  
  .config-grid {
    grid-template-columns: 1fr;
  }
  
  .service-controls .button-group {
    flex-direction: column;
  }
  
  .service-controls {
    gap: var(--spacing-md);
  }
  
  .service-controls .btn {
    min-width: 100%;
  }
  
  .api-endpoints-section .api-info li {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--spacing-xs);
  }
  
  .api-endpoints-section .api-info li strong {
    min-width: auto;
  }
}

@media (max-width: 480px) {
  .page-layout {
    padding: var(--spacing-sm);
  }
  
  .status-row {
    grid-template-columns: 1fr;
  }
}
</style>