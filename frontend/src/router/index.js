import { createRouter, createWebHistory } from 'vue-router'
import MainConfig from '@/views/MainConfig.vue'
import ServerConfig from '@/views/ServerConfig.vue'
import LoggingConfig from '@/views/LoggingConfig.vue'
import Copyright from '@/views/Copyright.vue'

const routes = [
  { path: '/', redirect: '/main' },
  { path: '/main', name: 'MainConfig', component: MainConfig },
  { path: '/server', name: 'ServerConfig', component: ServerConfig },
  { path: '/logging', name: 'LoggingConfig', component: LoggingConfig },
  { path: '/copyright', name: 'Copyright', component: Copyright }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router