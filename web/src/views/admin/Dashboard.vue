<template>
  <div class="dashboard-brutal">
    <!-- KPI Row -->
    <div class="kpi-row">
      <div class="kpi-card">
        <div class="kpi-label">总分享数</div>
        <div class="kpi-val">{{ stats.total_shares || 0 }}</div>
      </div>
      <div class="kpi-card" style="background: var(--acid-green);">
        <div class="kpi-label">总访问量</div>
        <div class="kpi-val">{{ stats.total_views || 0 }}</div>
      </div>
      <div class="kpi-card">
        <div class="kpi-label">总下载量</div>
        <div class="kpi-val">{{ stats.total_downloads || 0 }}</div>
      </div>
      <div class="kpi-card" style="background: var(--acid-pink); color: white;">
        <div class="kpi-label">今日访问</div>
        <div class="kpi-val">{{ stats.today_views || 0 }}</div>
      </div>
    </div>

    <!-- Console / Actions Area -->
    <div class="console-grid">
      <!-- Quick Actions -->
      <div class="console-block">
        <div class="console-header">> 快捷操作</div>
        <div class="console-content">
          <button class="btn-brutal smaller" @click="router.push('/admin/files')">浏览文件</button>
          <button class="btn-brutal smaller" @click="router.push('/admin/shares')">管理分享</button>
          <button class="btn-brutal smaller" @click="router.push('/admin/settings')">系统配置</button>
        </div>
      </div>

      <!-- System Status -->
      <div class="console-block">
        <div class="console-header">> 系统状态</div>
        <div class="console-content terminal-font">
          <div>状态: <span :style="{ color: initialized ? 'green' : 'red' }">{{ initialized ? '运行中' : '离线' }}</span></div>
          <div>-----------------</div>
          <div>节点: {{ siteTitle || 'WOOPEN_CORE' }}</div>
          <div>TOKEN: {{ initialized ? '已验证' : '丢失' }}</div>
          <div v-if="!initialized" style="margin-top: 10px;">
             <button class="btn-brutal smaller" @click="router.push('/admin/settings')" style="background: red; color: white;">立即修复</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { statsApi, settingsApi } from '../../api'

const router = useRouter()

const stats = ref<Record<string, number>>({})
const initialized = ref(false)
const siteTitle = ref('')

async function loadData() {
  try {
    const [statsRes, settingsRes] = await Promise.all([
      statsApi.overview(),
      settingsApi.get()
    ])
    stats.value = statsRes.data
    initialized.value = settingsRes.data.initialized
    siteTitle.value = settingsRes.data.site_title
  } catch (error) {
    // Error handled
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.dashboard-brutal {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* KPI Grid */
.kpi-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 15px;
}

.kpi-card {
  background: white;
  border: var(--border-thin);
  padding: 15px;
  box-shadow: 4px 4px 0 black;
  transition: transform 0.1s;
}

.kpi-card:hover {
  transform: translateY(-2px);
  box-shadow: 6px 6px 0 black;
}

.kpi-val { 
  font-size: 32px; 
  font-weight: 900; 
  font-family: 'Helvetica Neue', sans-serif;
  margin-top: 5px;
}

.kpi-label { 
  font-size: 12px; 
  text-transform: uppercase; 
  background: black; 
  color: white; 
  padding: 2px 5px; 
  display: inline-block; 
}

/* Console Grid */
.console-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.console-block {
  border: var(--border-thick);
  background: white;
}

.console-header {
  background: black;
  color: white;
  padding: 10px;
  font-family: monospace;
  font-weight: bold;
}

.console-content {
  padding: 20px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.terminal-font {
  font-family: monospace;
  font-size: 14px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.btn-brutal.smaller {
  padding: 10px 15px;
  font-size: 14px;
}

/* Mobile */
@media (max-width: 768px) {
  .kpi-row {
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  
  .console-grid {
    grid-template-columns: 1fr;
  }
}
</style>
