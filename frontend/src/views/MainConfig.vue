<template>
  <div class="main-config">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>主要配置</span>
        </div>
      </template>
      
      <el-form :model="form" label-width="120px">
        <!-- Monica配置 -->
        <el-divider content-position="left">Monica配置</el-divider>
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
        
        <!-- 代理配置开关 -->
        <el-divider content-position="left">代理配置</el-divider>
        <el-form-item label="启用代理">
          <el-switch v-model="form.proxy.enabled" />
          <el-alert
            v-if="form.proxy.enabled && (!form.proxy.httpProxy && !form.proxy.httpsProxy)"
            title="代理已启用但未配置具体地址，请到'服务器配置'页面设置代理地址"
            type="warning"
            :closable="false"
            show-icon
            style="margin-top: 10px;"
          />
          <el-alert
            v-if="form.proxy.enabled && (form.proxy.httpProxy || form.proxy.httpsProxy)"
            :title="`代理已启用: ${form.proxy.httpProxy || form.proxy.httpsProxy}`"
            type="success"
            :closable="false"
            show-icon
            style="margin-top: 10px;"
          />
        </el-form-item>
        
        <!-- 安全配置 -->
        <el-divider content-position="left">安全配置</el-divider>
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
        
        <!-- 服务控制 -->
        <el-divider content-position="left">服务控制</el-divider>
        
        <el-form-item>
          <el-space>
            <el-button
              type="success"
              :disabled="appStore.isServiceRunning"
              @click="startService_btn"
              :loading="loading"
            >
              <el-icon><VideoPlay /></el-icon>
              启动服务
            </el-button>
            
            <el-button
              type="danger"
              :disabled="!appStore.isServiceRunning"
              @click="stopService_btn"
              :loading="loading"
            >
              <el-icon><VideoPause /></el-icon>
              停止服务
            </el-button>
            
            <el-button
              type="primary"
              :disabled="!appStore.isServiceRunning"
              @click="testConfig"
              :loading="loading"
            >
              <el-icon><Connection /></el-icon>
              测试 API 配置
            </el-button>
            
            <el-button
              type="info"
              @click="getQuota"
              :loading="loading"
            >
              <el-icon><Coin /></el-icon>
              查询 Monica 额度
            </el-button>
          </el-space>
        </el-form-item>
        
        <!-- 状态显示 -->
        <el-form-item>
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
        </el-form-item>
        
        <!-- API端点信息 -->
        <el-divider content-position="left">API端点信息</el-divider>
        <el-form-item>
          <div class="api-endpoints">
            <el-card>
              <ul>
                <li><strong>POST</strong> /v1/chat/completions - 聊天对话（兼容ChatGPT）</li>
                <li><strong>GET</strong> /v1/models - 获取模型列表</li>
                <li><strong>POST</strong> /v1/images/generations - 图片生成（兼容DALL-E）</li>
              </ul>
            </el-card>
          </div>
        </el-form-item>
      </el-form>
      
      <!-- 保存配置按钮 -->
      <div class="save-section">
        <el-button type="primary" @click="saveConfig" :loading="loading">
          <el-icon><Check /></el-icon>
          保存配置
        </el-button>
      </div>
    </el-card>
    
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
.main-config {
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

.status-info {
  width: 100%;
}

.api-info {
  margin-top: 10px;
  padding: 10px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.quota-info {
  margin-top: 10px;
}

.api-endpoints ul {
  list-style-type: none;
  padding: 0;
}

.api-endpoints li {
  padding: 8px 0;
  border-bottom: 1px solid #eee;
}

.api-endpoints li:last-child {
  border-bottom: none;
}

.save-section {
  text-align: center;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #eee;
}

.test-results {
  max-height: 70vh;
  overflow-y: auto;
}

.test-details {
  padding: 10px;
}

.test-details pre {
  background-color: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.test-details .error {
  background-color: #fef0f0;
  color: #f56c6c;
}

.result-success {
  color: #67c23a;
  font-weight: bold;
}

.result-error {
  color: #f56c6c;
  font-weight: bold;
}
</style>