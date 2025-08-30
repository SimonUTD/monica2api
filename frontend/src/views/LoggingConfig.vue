<template>
  <div class="logging-config">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>日志配置</span>
        </div>
      </template>
      
      <el-form :model="form" label-width="120px">
        <!-- 日志基本配置 -->
        <el-divider content-position="left">日志基本配置</el-divider>
        <el-form-item label="日志级别">
          <el-select v-model="form.logging.level" placeholder="选择日志级别">
            <el-option label="debug" value="debug" />
            <el-option label="info" value="info" />
            <el-option label="warn" value="warn" />
            <el-option label="error" value="error" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="日志格式">
          <el-select v-model="form.logging.format" placeholder="选择日志格式">
            <el-option label="json" value="json" />
            <el-option label="console" value="console" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="日志输出">
          <el-select v-model="form.logging.output" placeholder="选择日志输出方式">
            <el-option label="stdout" value="stdout" />
            <el-option label="stderr" value="stderr" />
            <el-option label="file" value="file" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="启用请求日志">
          <el-switch v-model="form.logging.enableRequestLog" />
        </el-form-item>
        
        <el-form-item label="掩盖敏感信息">
          <el-switch v-model="form.logging.maskSensitive" />
        </el-form-item>
        
        <!-- 日志文件信息 -->
        <el-divider content-position="left">日志文件信息</el-divider>
        <el-form-item>
          <el-card>
            <template #header>
              <span>日志文件路径</span>
            </template>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="日志文件路径">
                <el-input v-model="logFilePath" readonly>
                  <template #append>
                    <el-button @click="openLogDirectory">
                      <el-icon><FolderOpened /></el-icon>
                      打开目录
                    </el-button>
                  </template>
                </el-input>
              </el-descriptions-item>
              <el-descriptions-item label="日志级别">
                <el-tag :type="getLogLevelTag(form.logging.level)">
                  {{ form.logging.level }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="日志格式">
                <el-tag>{{ form.logging.format }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="日志输出">
                <el-tag>{{ form.logging.output }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="请求日志">
                <el-tag :type="form.logging.enableRequestLog ? 'success' : 'info'">
                  {{ form.logging.enableRequestLog ? '启用' : '禁用' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="敏感信息掩盖">
                <el-tag :type="form.logging.maskSensitive ? 'success' : 'info'">
                  {{ form.logging.maskSensitive ? '启用' : '禁用' }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </el-card>
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {UpdateConfig,GetConfig,OpenLogDirectory,GetLogFilePath} from '../../wailsjs/wailsjs/go/main/WailsApp.js'
const form = reactive({
  logging: {
    level: 'info',
    format: 'json',
    output: 'file',
    enableRequestLog: true,
    maskSensitive: true
  }
})

const logFilePath = ref('~/.monica-proxy/logs/monica-proxy.log')
const loading = ref(false)

onMounted(async () => {
  await loadConfig()
  // 获取实际的日志文件路径
  try {
    const actualPath = await GetLogFilePath()
    logFilePath.value = actualPath
  } catch (error) {
    console.log('获取日志路径失败:', error)
  }
})

async function loadConfig() {
  try {
    const config = await GetConfig()
    if (config.logging) {
      Object.assign(form.logging, config.logging)
    }
  } catch (error) {
    ElMessage.error('加载配置失败: ' + error.message)
  }
}

async function saveConfig() {
  loading.value = true
  try {
    await UpdateConfig({
      logging: form.logging
    })
    ElMessage.success('配置保存成功')
  } catch (error) {
    ElMessage.error('配置保存失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

function getLogLevelTag(level) {
  switch (level) {
    case 'debug':
      return 'info'
    case 'info':
      return 'success'
    case 'warn':
      return 'warning'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
}

async function openLogDirectory() {
  try {
    await OpenLogDirectory()
    ElMessage.success('已打开日志文件目录')
  } catch (error) {
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('打开目录失败: ' + errorMsg)
  }
}
</script>

<style scoped>
.logging-config {
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