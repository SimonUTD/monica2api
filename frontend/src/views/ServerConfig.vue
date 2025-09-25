<template>
  <div class="page-layout compact">
    <!-- 状态卡片区域 -->
    <div class="status-row">
      <div class="status-card">
        <div class="status-icon success">
          <el-icon><Monitor /></el-icon>
        </div>
        <h3 class="status-title">服务器状态</h3>
        <p class="status-description">
          {{ form.server.host }}:{{ form.server.port }}
        </p>
      </div>
      
      <div class="status-card" :class="hasProxy ? 'success' : 'info'">
        <div class="status-icon" :class="hasProxy ? 'success' : 'info'">
          <el-icon><Share /></el-icon>
        </div>
        <h3 class="status-title">代理状态</h3>
        <p class="status-description">
          {{ proxyStatus }}
        </p>
      </div>
      
      <div class="status-card">
        <div class="status-icon info">
          <el-icon><Timer /></el-icon>
        </div>
        <h3 class="status-title">超时设置</h3>
        <p class="status-description">
          读取: {{ form.server.readTimeout }}s | 写入: {{ form.server.writeTimeout }}s
        </p>
      </div>
    </div>
    
    <!-- 配置表单区域 -->
    <div class="config-grid">
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><Monitor /></el-icon>
          <div>
            <h3 class="config-card-title">服务器基本配置</h3>
            <p class="config-card-description">配置服务器监听地址和端口参数</p>
          </div>
        </div>
        
        <el-form :model="form" label-width="120px" size="large" class="form-large">
        <el-form-item label="主机地址">
          <el-input
            v-model="form.server.host"
            placeholder="服务器监听的主机地址"
          />
        </el-form-item>
        
        <el-form-item label="端口">
          <el-input-number
            v-model="form.server.port"
            :min="1"
            :max="65535"
          />
        </el-form-item>
        
        <el-form-item label="读取超时(秒)">
          <el-input-number
            v-model="form.server.readTimeout"
            :min="1"
            :max="300"
          />
        </el-form-item>
        
        <el-form-item label="写入超时(秒)">
          <el-input-number
            v-model="form.server.writeTimeout"
            :min="1"
            :max="300"
          />
        </el-form-item>
        
        <el-form-item label="空闲超时(秒)">
          <el-input-number
            v-model="form.server.idleTimeout"
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
            <p class="config-card-description">配置HTTP/HTTPS代理设置</p>
          </div>
        </div>
        
        <el-form :model="form" label-width="120px" size="large" class="form-large">
        <el-form-item label="HTTP代理">
          <el-input
            v-model="form.proxy.httpProxy"
            placeholder="HTTP代理地址（例如：http://proxy.example.com:8080）"
          />
        </el-form-item>
        
        <el-form-item label="HTTPS代理">
          <el-input
            v-model="form.proxy.httpsProxy"
            placeholder="HTTPS代理地址（例如：https://proxy.example.com:8080）"
          />
        </el-form-item>
        
        <el-form-item label="不使用代理">
          <el-input
            v-model="form.proxy.noProxy"
            placeholder="不使用代理的域名列表（逗号分隔）"
          />
        </el-form-item>
        
        <!-- 代理状态显示 -->
        <el-divider content-position="left">代理状态</el-divider>
        <el-form-item>
          <el-alert
            :title="proxyStatus"
            :type="hasProxy ? 'warning' : 'info'"
            :closable="false"
            show-icon
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
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {UpdateConfig,GetConfig} from '../../wailsjs/wailsjs/go/main/WailsApp.js'

const form = reactive({
  server: {
    host: '0.0.0.0',
    port: 8080,
    readTimeout: 30,
    writeTimeout: 30,
    idleTimeout: 60
  },
  proxy: {
    httpProxy: '',
    httpsProxy: '',
    noProxy: ''
  }
})

const loading = ref(false)

const hasProxy = computed(() => {
  return form.proxy.httpProxy || form.proxy.httpsProxy
})

const proxyStatus = computed(() => {
  if (hasProxy.value) {
    return '已启用代理'
  }
  return '未启用代理'
})

onMounted(async () => {
  await loadConfig()
})

async function loadConfig() {
  try {
    const config = await GetConfig()
    if (config.server) {
      Object.assign(form.server, config.server)
    }
    if (config.proxy) {
      Object.assign(form.proxy, config.proxy)
    }
  } catch (error) {
    ElMessage.error('加载配置失败: ' + error.message)
  }
}

async function saveConfig() {
  loading.value = true
  try {
    await UpdateConfig({
      server: form.server,
      proxy: form.proxy
    })
    ElMessage.success('配置保存成功')
  } catch (error) {
    ElMessage.error('配置保存失败: ' + error.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* 页面布局增强 */
.page-layout {
  padding: var(--spacing-sm);
  background: var(--background-page);
  min-height: 100vh;
}

.compact {
  /* 紧凑模式：无顶部标题区域 */
}

/* 状态卡片样式增强 */
.status-row {
  margin-bottom: var(--spacing-md);
}

.status-card {
  transition: transform var(--transition-normal);
}

.status-card:hover {
  transform: translateY(-4px);
}

/* 配置网格布局 */
.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: var(--card-gap);
  margin-bottom: var(--spacing-md);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-layout {
    padding: var(--spacing-sm);
  }
  
  .config-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 480px) {
  .page-layout {
    padding: var(--spacing-xs);
  }
  
  .status-row {
    grid-template-columns: 1fr;
  }
}
</style>