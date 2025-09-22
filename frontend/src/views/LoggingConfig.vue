<template>
  <div class="logging-config">
    <!-- 日志系统状态卡片 -->
    <el-row :gutter="20" class="status-row">
      <el-col :span="8">
        <el-card shadow="hover" class="status-card">
          <div class="status-item">
            <el-icon class="status-icon" :class="getLogLevelClass(form.logging.level)">
              <Document />
            </el-icon>
            <div class="status-info">
              <div class="status-title">日志级别</div>
              <div class="status-value">
                <el-tag :type="getLogLevelTag(form.logging.level)" size="large">
                  {{ form.logging.level.toUpperCase() }}
                </el-tag>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="status-card">
          <div class="status-item">
            <el-icon class="status-icon success">
              <FolderOpened />
            </el-icon>
            <div class="status-info">
              <div class="status-title">日志文件</div>
              <div class="status-value">{{ formatFileSize(logFileSize) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="status-card">
          <div class="status-item">
            <el-icon class="status-icon" :class="form.logging.enableRequestLog ? 'warning' : 'info'">
              <Warning />
            </el-icon>
            <div class="status-info">
              <div class="status-title">详细日志</div>
              <div class="status-value">
                <el-tag :type="form.logging.enableRequestLog ? 'warning' : 'info'" size="large">
                  {{ form.logging.enableRequestLog ? '已启用' : '已禁用' }}
                </el-tag>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 主要配置区域 -->
    <el-row :gutter="20">
      <el-col :span="12">
        <!-- 基本配置卡片 -->
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <el-icon><Setting /></el-icon>
              <span>基本配置</span>
            </div>
          </template>
          
          <el-form :model="form" label-width="100px" size="large">
            <el-form-item label="日志级别">
              <el-select v-model="form.logging.level" placeholder="选择日志级别" style="width: 100%">
                <el-option label="DEBUG - 调试信息" value="debug" />
                <el-option label="INFO - 一般信息" value="info" />
                <el-option label="WARN - 警告信息" value="warn" />
                <el-option label="ERROR - 错误信息" value="error" />
              </el-select>
            </el-form-item>
            
            <el-form-item label="日志格式">
              <el-select v-model="form.logging.format" placeholder="选择日志格式" style="width: 100%">
                <el-option label="JSON 格式" value="json" />
                <el-option label="控制台格式" value="console" />
              </el-select>
            </el-form-item>
            
            <el-form-item label="输出方式">
              <el-input 
                v-model="outputDisplay" 
                readonly 
                placeholder="文件输出"
              >
                <template #prefix>
                  <el-icon><Document /></el-icon>
                </template>
              </el-input>
            </el-form-item>
            
            <el-form-item label="敏感信息">
              <el-switch 
                v-model="form.logging.maskSensitive"
                active-text="掩盖"
                inactive-text="不掩盖"
                size="large"
              />
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
      
      <el-col :span="12">
        <!-- 高级配置卡片 -->
        <el-card class="config-card">
          <template #header>
            <div class="card-header">
              <el-icon><Tools /></el-icon>
              <span>高级配置</span>
            </div>
          </template>
          
          <el-form :model="form" label-width="100px" size="large">
            <el-form-item label="详细日志">
              <el-switch 
                v-model="form.logging.enableRequestLog"
                active-text="启用"
                inactive-text="禁用"
                size="large"
                @change="onRequestLogChange"
              />
              <el-button 
                v-if="form.logging.enableRequestLog"
                type="text" 
                @click="showDetails = !showDetails"
                style="margin-left: 10px;"
              >
                <el-icon><View /></el-icon>
                {{ showDetails ? '收起详情' : '查看详情' }}
              </el-button>
            </el-form-item>
            
            <!-- 可折叠的详细信息 -->
            <el-collapse-transition>
              <div v-show="showDetails && form.logging.enableRequestLog" class="details-alert">
                <el-alert 
                  type="warning" 
                  :closable="false"
                  title="详细日志说明"
                >
                  <template #default>
                    <div class="details-content">
                      <p><strong>详细请求日志会记录以下内容：</strong></p>
                      <ul>
                        <li>外部工具调用本软件的请求详情（环节1）</li>
                        <li>本软件请求Monica API的详情（环节2）</li>
                        <li>Monica返回本软件的响应详情（环节3）</li>
                        <li>本软件返回外部工具的响应详情（环节4）</li>
                      </ul>
                      <p class="warning-text">
                        <el-icon><WarningFilled /></el-icon>
                        建议仅在调试问题时启用，日常使用请保持禁用状态
                      </p>
                    </div>
                  </template>
                </el-alert>
              </div>
            </el-collapse-transition>
            
            <el-divider />
            
            <!-- 日志文件管理 -->
            <div class="file-management">
              <el-form-item label="日志文件路径">
                <el-input v-model="logFilePath" readonly size="small">
                    <template #append>
                      <el-button @click="openLogDirectory" size="small">
                        <el-icon><FolderOpened /></el-icon>
                      </el-button>
                    </template>
                  </el-input>
            </el-form-item>
            <el-form-item label="日志文件大小">
              <el-input v-model="logFileSize" readonly size="small">
                    <template #append>
                      <el-button 
                        @click="clearLogFile" 
                        type="danger" 
                        size="small"
                        :disabled="logFileSize === '文件不存在' || logFileSize === '0 B'"
                      >
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-input>
            </el-form-item>
              
            </div>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
    
    <!-- 保存配置按钮 -->
    <div class="save-section">
      <el-button 
        type="primary" 
        @click="saveConfig" 
        :loading="loading"
        size="large"
      >
        <el-icon><Check /></el-icon>
        保存配置
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, FolderOpened, Delete, Setting, Tools, Document, Warning, WarningFilled } from '@element-plus/icons-vue'
import {UpdateConfig,GetConfig,OpenLogDirectory,GetLogFilePath,GetLogFileSize,ClearLogFile} from '../../wailsjs/wailsjs/go/main/WailsApp.js'
const form = reactive({
  logging: {
    level: 'info',
    format: 'json',
    output: 'file', // 固定为文件输出，编译后无法更改
    enableRequestLog: false, // 默认禁用详细请求日志，防止日志爆炸
    maskSensitive: true
  }
})

const logFilePath = ref('~/.monica-proxy/logs/monica-proxy.log')
const logFileSize = ref('计算中...')
const loading = ref(false)
const showDetails = ref(false)
const outputDisplay = ref('文件输出')

onMounted(async () => {
  await loadConfig()
  // 获取实际的日志文件路径和大小
  await updateLogFileInfo()
})

async function updateLogFileInfo() {
  try {
    const actualPath = await GetLogFilePath()
    logFilePath.value = actualPath
    
    const size = await GetLogFileSize()
    logFileSize.value = size
  } catch (error) {
    console.log('获取日志文件信息失败:', error)
  }
}

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

function getLogLevelClass(level) {
  switch (level) {
    case 'debug':
      return 'debug'
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

function formatFileSize(size) {
  if (size === '文件不存在' || size === '0 B') {
    return size
  }
  return size
}

function onRequestLogChange(value) {
  if (value) {
    // 启用详细日志时自动显示详情
    showDetails.value = true
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

async function clearLogFile() {
  try {
    await ElMessageBox.confirm(
      '确定要清空日志文件内容吗？此操作不可恢复，所有日志记录将被永久删除。',
      '确认清空日志',
      {
        confirmButtonText: '确定清空',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: true,
      }
    )
    
    await ClearLogFile()
    await updateLogFileInfo() // 更新文件大小显示
    ElMessage.success('日志文件已清空')
  } catch (error) {
    if (error === 'cancel') {
      // 用户取消操作，不显示错误
      return
    }
    const errorMsg = error?.message || error?.toString() || '未知错误'
    ElMessage.error('清空日志文件失败: ' + errorMsg)
  }
}
</script>

<style scoped>
.logging-config {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

/* 状态卡片样式 */
.status-row {
  margin-bottom: 30px;
}

.status-card {
  border: none;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.status-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.status-item {
  display: flex;
  align-items: center;
  gap: 15px;
}

.status-icon {
  font-size: 32px;
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.status-icon.success {
  background: linear-gradient(135deg, #84fab0 0%, #8fd3f4 100%);
  color: white;
}

.status-icon.debug {
  background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
  color: #666;
}

.status-icon.warning {
  background: linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%);
  color: #e6a23c;
}

.status-icon.danger {
  background: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%);
  color: #f56c6c;
}

.status-info {
  flex: 1;
}

.status-title {
  font-size: 14px;
  color: #666;
  margin-bottom: 5px;
}

.status-value {
  font-size: 16px;
  font-weight: bold;
  color: #333;
}

/* 配置卡片样式 */
.config-card {
  border: none;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  margin-bottom: 20px;
  transition: all 0.3s ease;
}

.config-card:hover {
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: bold;
  font-size: 16px;
  color: #333;
}

.card-header .el-icon {
  font-size: 20px;
  color: #409EFF;
}

.header-actions {
  margin-left: auto;
}

/* 详细信息样式 */
.details-alert {
  margin-top: 15px;
}

.details-content {
  font-size: 14px;
  line-height: 1.6;
}

.details-content ul {
  margin: 10px 0;
  padding-left: 20px;
}

.details-content li {
  margin-bottom: 5px;
}

.warning-text {
  color: #e6a23c;
  font-weight: bold;
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 5px;
}

/* 文件管理样式 */
.file-management {
  margin-top: 15px;
}

.file-management h4 {
  margin: 0 0 10px 0;
  color: #333;
  font-size: 14px;
}

/* 帮助文本样式 */
.help-text {
  margin-top: 8px;
  font-size: 14px;
}

.help-text .el-alert {
  margin-bottom: 0;
}


/* 保存按钮样式 */
.save-section {
  text-align: center;
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px solid #eee;
}

.save-section .el-button {
  min-width: 120px;
  height: 40px;
  font-size: 16px;
  font-weight: bold;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .logging-config {
    padding: 10px;
  }
  
  .status-row .el-col {
    margin-bottom: 15px;
  }
  
  .config-card {
    margin-bottom: 15px;
  }
  
  .status-item {
    flex-direction: column;
    text-align: center;
    gap: 10px;
  }
  
  .status-icon {
    width: 50px;
    height: 50px;
    font-size: 24px;
  }
}

/* 动画效果 */
.el-card {
  transition: all 0.3s ease;
}

.el-card:hover {
  transform: translateY(-2px);
}

.el-tag {
  font-weight: 500;
}

.el-button-group .el-button {
  margin-left: 0;
}

.el-button-group .el-button:not(:first-child) {
  margin-left: -1px;
}

/* 图标颜色 */
.el-icon {
  transition: color 0.3s ease;
}

.status-icon.success {
  color: #67c23a;
}

.status-icon.debug {
  color: #909399;
}

.status-icon.warning {
  color: #e6a23c;
}

.status-icon.danger {
  color: #f56c6c;
}
</style>