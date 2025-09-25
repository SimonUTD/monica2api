<template>
  <div class="page-layout compact">
    <!-- 日志系统状态卡片 -->
    <div class="status-row">
      <div class="status-card">
        <div class="status-icon" :class="getLogLevelClass(form.logging.level)">
          <el-icon><Document /></el-icon>
        </div>
        <h3 class="status-title">日志级别</h3>
        <p class="status-description">
          <el-tag :type="getLogLevelTag(form.logging.level)" size="large">
            {{ form.logging.level.toUpperCase() }}
          </el-tag>
        </p>
      </div>
      
      <div class="status-card success">
        <div class="status-icon success">
          <el-icon><FolderOpened /></el-icon>
        </div>
        <h3 class="status-title">日志文件</h3>
        <p class="status-description">{{ formatFileSize(logFileSize) }}</p>
      </div>
      
      <div class="status-card" :class="form.logging.enableRequestLog ? 'warning' : 'info'">
        <div class="status-icon" :class="form.logging.enableRequestLog ? 'warning' : 'info'">
          <el-icon><Warning /></el-icon>
        </div>
        <h3 class="status-title">详细日志</h3>
        <p class="status-description">
          <el-tag :type="form.logging.enableRequestLog ? 'warning' : 'info'" size="large">
            {{ form.logging.enableRequestLog ? '已启用' : '已禁用' }}
          </el-tag>
        </p>
      </div>
    </div>

    <!-- 配置表单区域 -->
    <div class="config-grid">
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><Setting /></el-icon>
          <div>
            <h3 class="config-card-title">基本配置</h3>
            <p class="config-card-description">配置日志级别、格式等基本参数</p>
          </div>
        </div>
        
        <el-form :model="form" label-width="120px" size="large" class="form-large">
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
      </div>
      
      <div class="config-card">
        <div class="config-card-header">
          <el-icon><Tools /></el-icon>
          <div>
            <h3 class="config-card-title">高级配置</h3>
            <p class="config-card-description">详细日志和文件管理设置</p>
          </div>
        </div>
        
        <el-form :model="form" label-width="120px" size="large" class="form-large">
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
              class="ml-sm"
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

/* 调试级别特殊样式 */
.status-icon.debug {
  background: var(--gradient-info);
  color: white;
}

/* 危险级别特殊样式 */
.status-icon.danger {
  background: var(--gradient-error);
  color: white;
}

/* 配置网格布局 */
.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: var(--card-gap);
  margin-bottom: var(--spacing-xl);
}

/* 详细信息样式 */
.details-alert {
  margin-top: var(--spacing-md);
}

.details-content {
  font-size: var(--font-size-sm);
  line-height: var(--line-height-normal);
}

.details-content ul {
  margin: var(--spacing-sm) 0;
  padding-left: var(--spacing-lg);
}

.details-content li {
  margin-bottom: var(--spacing-xs);
}

.warning-text {
  color: var(--warning-color);
  font-weight: var(--font-weight-bold);
  margin-top: var(--spacing-sm);
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
}

/* 文件管理样式 */
.file-management {
  margin-top: var(--spacing-md);
}

/* 表单样式增强 */
.form-large {
  margin-top: var(--spacing-md);
}

.form-large .el-form-item {
  margin-bottom: var(--spacing-lg);
}

/* 按钮样式统一 */
.ml-sm {
  margin-left: var(--spacing-sm);
}

.mt-sm {
  margin-top: var(--spacing-sm);
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