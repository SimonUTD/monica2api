import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAppStore = defineStore('app', () => {
  const config = ref({})
  const serviceStatus = ref({
    isRunning: false,
    message: '服务未启动',
    address: '',
    apiKey: ''
  })
  const loading = ref(false)
  const testResults = ref([])
  const quotaInfo = ref({})

  const isServiceRunning = computed(() => serviceStatus.value.isRunning)

  return {
    config,
    serviceStatus,
    loading,
    testResults,
    quotaInfo,
    isServiceRunning
  }
})