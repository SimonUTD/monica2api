<template>
  <div class="server-config">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>服务器配置</span>
        </div>
      </template>
      
      <el-form :model="form" label-width="120px">
        <!-- 服务器基本配置 -->
        <el-divider content-position="left">服务器基本配置</el-divider>
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
        
        <!-- 代理配置 -->
        <el-divider content-position="left">代理配置</el-divider>
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
      
      <!-- 保存配置按钮 -->
      <div class="save-section">
        <el-button type="primary" @click="saveConfig" :loading="loading">
          <el-icon><Check /></el-icon>
          保存配置
        </el-button>
      </div>
    </el-card>
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
.server-config {
  max-width: 800px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: bold;
  font-size: 18px;
}

.save-section {
  text-align: center;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #eee;
}
</style>