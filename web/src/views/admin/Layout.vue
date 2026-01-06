<template>
  <div class="admin-container">
    <div class="window-frame active admin-window">
      <!-- Window Header -->
      <div class="window-header">
        <span>系统管理控制台</span>
        <div class="window-controls">
          <button @click="minimize">_</button>
          <button @click="close">X</button>
        </div>
      </div>

      <div class="dash-container">
        <!-- Sidebar -->
        <div class="sidebar">
          <button 
            v-for="item in menuItems" 
            :key="item.path"
            class="nav-btn sound-click"
            :class="{ active: currentRoute === item.path }"
            @click="navigate(item.path)"
          >
            {{ item.label }}
          </button>
          
          <button class="nav-btn sound-click terminate-btn" @click="handleLogout">
            退出系统
          </button>
          
          <div class="sys-stats">
            RAM: 64TB<br>CPU: 99%<br>PING: 1ms<br>
            USER: ADMIN
          </div>
        </div>

        <!-- Main Content -->
        <div class="main-dash">
          <router-view v-slot="{ Component }">
            <transition name="fade" mode="out-in">
              <component :is="Component" />
            </transition>
          </router-view>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const currentRoute = computed(() => route.path)

const menuItems = [
  { path: '/admin/dashboard', label: '概览监控' },
  { path: '/admin/files', label: '文件浏览' },
  { path: '/admin/shares', label: '分享管理' },
  { path: '/admin/settings', label: '系统设置' }
]

function navigate(path: string) {
  router.push(path)
}

function handleLogout() {
  ElMessageBox.confirm(
    '确定要断开与系统的连接吗？',
    '警告: 连接终止',
    {
      confirmButtonText: '立即断开',
      cancelButtonText: '取消',
      type: 'warning',
      customClass: 'brutal-modal'
    }
  ).then(() => {
    authStore.logout()
    ElMessage.success('已安全退出')
    router.push('/admin/login')
  })
}

function minimize() {
  ElMessage.info('系统核心进程不可最小化')
}

function close() {
  handleLogout()
}
</script>

<style scoped>
.admin-container {
  height: 100vh;
  padding: 20px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.admin-window {
  width: 1200px; /* Wider for admin panel */
  height: 800px;
  max-width: 100%;
  max-height: 100%;
  display: flex; /* Override hidden */
  flex-direction: column;
}

.dash-container {
  display: grid;
  grid-template-columns: 220px 1fr;
  flex: 1; /* Take remaining height */
  overflow: hidden; /* Contain inner scrolling */
}

/* Sidebar Styling */
.sidebar {
  background: var(--pure-black);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 15px;
  border-right: var(--border-thick);
}

.nav-btn {
  background: transparent;
  border: 2px solid #333;
  color: #888;
  padding: 15px;
  text-align: left;
  font-family: 'Helvetica Neue', sans-serif; /* Clean font for Chinese */
  font-weight: bold;
  font-size: 16px;
  transition: 0.2s;
  cursor: pointer;
  text-transform: uppercase;
}

.nav-btn:hover {
  border-color: var(--acid-green);
  color: var(--pure-white);
}

.nav-btn.active {
  background: var(--acid-green);
  color: black;
  border-color: var(--acid-green);
  box-shadow: 4px 4px 0 white;
}

.terminate-btn {
  margin-top: auto; /* Push to bottom before stats */
  color: #f56c6c; /* Redish */
  border-color: #f56c6c;
}

.terminate-btn:hover {
  background: #f56c6c;
  color: white;
  border-color: #f56c6c;
  box-shadow: 4px 4px 0 white;
}

.sys-stats {
  margin-top: 20px;
  color: white;
  font-family: monospace;
  font-size: 10px;
  opacity: 0.7;
  line-height: 1.5;
}

/* Main Dash Content */
.main-dash {
  padding: 20px;
  background: #f0f0f0;
  overflow-y: auto; 
  display: flex;
  flex-direction: column;
}

.window-controls button {
    background: var(--acid-pink);
    border: 2px solid white;
    color: white;
    width: 25px;
    height: 25px;
    font-weight: bold;
    line-height: 20px;
    margin-left: 5px;
    box-shadow: 2px 2px 0 black;
    cursor: pointer;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .admin-container {
    padding: 0;
  }
  
  .admin-window {
    width: 100%;
    height: 100vh;
    border: none;
  }
  
  .dash-container {
    display: flex;
    flex-direction: column;
  }
  
  .sidebar {
    flex-direction: row;
    padding: 10px;
    overflow-x: auto;
    border-right: none;
    border-bottom: var(--border-thick);
    gap: 10px;
    flex-shrink: 0;
  }
  
  .nav-btn {
    padding: 8px 12px;
    font-size: 14px;
    white-space: nowrap;
    margin-top: 0 !important; /* Override margin-top auto */
  }
  
  .sys-stats {
    display: none;
  }
  
  .main-dash {
    padding: 10px;
  }

  .window-header {
    font-size: 14px;
  }
}
</style>
