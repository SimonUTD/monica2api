<template>
  <div id="app">
    <el-container style="height: 100vh;">
      <el-header style="background-color: #409EFF; color: white; display: flex; align-items: center; justify-content: space-between;">
        <h2 style="margin: 0;">Monica Proxy Wails - 基于界面的Monica（Web）转换成 API 的工具</h2>
        <div>
          <el-tag v-if="appStore.isServiceRunning" type="success" effect="dark">
            <el-icon><VideoPlay /></el-icon>
            服务运行中
          </el-tag>
          <el-tag v-else type="info" effect="dark">
            <el-icon><VideoPause /></el-icon>
            服务未启动
          </el-tag>
        </div>
      </el-header>
      
      <el-container>
        <el-aside width="200px" style="background-color: #f5f5f5;">
          <el-menu
            :default-active="$route.path"
            router
            style="height: 100%; border-right: none;"
          >
            <el-menu-item index="/main">
              <el-icon><Setting /></el-icon>
              主要配置
            </el-menu-item>
            <el-menu-item index="/server">
              <el-icon><Server /></el-icon>
              服务器配置
            </el-menu-item>
            <el-menu-item index="/logging">
              <el-icon><Document /></el-icon>
              日志配置
            </el-menu-item>
            <el-menu-item index="/copyright">
              <el-icon><InfoFilled /></el-icon>
              版权信息
            </el-menu-item>
          </el-menu>
        </el-aside>
        
        <el-main>
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { useAppStore } from '@/stores/app'
import { onMounted } from 'vue'
import {GetServiceStatus,GetConfig} from '../wailsjs/wailsjs/go/main/WailsApp.js'
const appStore = useAppStore()

onMounted(async () => {
  await loadConfig()
  await getServiceStatu()
})

async function loadConfig() {
  try {
    const config = await GetConfig()
    appStore.config = config
  } catch (error) {
    console.error('加载配置失败:', error)
  }
}

async function getServiceStatu() {
  try {
    const status = await GetServiceStatus()
    appStore.serviceStatus = status
  } catch (error) {
    console.error('获取服务状态失败:', error)
  }
}
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  height: 100vh;
}

body {
  margin: 0;
  padding: 0;
}
</style>