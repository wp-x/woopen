<template>
  <div class="settings-page">
    <!-- 瀑布流布局容器 -->
    <div class="masonry-container">
      <!-- 云盘配置 -->
      <el-card class="masonry-item">
        <template #header>
          <div class="card-header">
            <span>云盘配置</span>
            <el-tag :type="settings.initialized ? 'success' : 'warning'" size="small">
              {{ settings.initialized ? '已连接' : '未连接' }}
            </el-tag>
          </div>
        </template>

        <el-form :model="cloudForm" label-width="100px" size="small">
          <el-form-item label="Refresh Token">
            <el-input
              v-model="cloudForm.refresh_token"
              type="password"
              show-password
              placeholder="从浏览器开发者工具获取"
            />
            <div class="form-tip">
              登录 <a href="https://pan.wo.cn" target="_blank">联通云盘</a> 后，在开发者工具 Network 中找到 refresh_token
            </div>
          </el-form-item>
          <el-form-item label="Access Token">
            <el-input
              v-model="cloudForm.access_token"
              type="password"
              show-password
              placeholder="与 Refresh Token 一起使用"
            />
            <div class="form-tip">
              可选：如果只拿到一个 Token，填写在 Refresh Token，Access Token 留空即可
            </div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="testToken" :loading="testing">
              测试连接
            </el-button>
            <el-button type="success" @click="saveCloudSettings" :loading="saving">
              保存配置
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 站点设置 -->
      <el-card class="masonry-item">
        <template #header>
          <span class="card-header">站点设置</span>
        </template>

        <el-form :model="siteForm" label-width="100px" size="small">
          <el-form-item label="站点标题">
            <el-input v-model="siteForm.site_title" placeholder="WoOpen" />
          </el-form-item>
          <el-form-item label="站点 Logo">
            <el-input v-model="siteForm.site_logo" placeholder="Logo URL（可选）" />
          </el-form-item>

          <el-divider content-position="left">登录界面自定义</el-divider>

          <el-form-item label="窗口标题">
            <el-input v-model="siteForm.login_title" placeholder="WoOpen_Auth_v10.exe" />
          </el-form-item>
          <el-form-item label="头像图标">
            <el-input v-model="siteForm.login_avatar" placeholder="💀 或图片URL" />
          </el-form-item>
          <el-form-item label="角色标签">
            <el-input v-model="siteForm.login_role_tag" placeholder="ROLE: ADMIN" />
          </el-form-item>
          <el-form-item label="等级标签">
            <el-input v-model="siteForm.login_level_tag" placeholder="LEVEL: 99" />
          </el-form-item>
          <el-form-item label="系统名称">
            <el-input v-model="siteForm.login_system_name" placeholder="WOOPEN CLOUD SYSTEM" />
          </el-form-item>

          <el-divider content-position="left">分享界面自定义</el-divider>

          <el-form-item label="底部署名">
            <el-input v-model="siteForm.share_footer" placeholder="Powered by WOOPEN_OS" />
          </el-form-item>

          <el-form-item>
            <el-button type="success" @click="saveSiteSettings" :loading="saving">
              保存站点设置
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 修改密码 -->
      <el-card class="masonry-item">
        <template #header>
          <span class="card-header">修改密码</span>
        </template>

        <el-form :model="passwordForm" label-width="80px" size="small">
          <el-form-item label="原密码">
            <el-input v-model="passwordForm.old_password" type="password" show-password />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="passwordForm.new_password" type="password" show-password />
          </el-form-item>
          <el-form-item label="确认密码">
            <el-input v-model="passwordForm.confirm_password" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="danger" @click="updatePassword" :loading="updatingPwd">
              修改密码
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 账号监控 -->
      <el-card class="masonry-item">
        <template #header>
          <div class="card-header">
            <span>账号监控</span>
            <el-tag :type="monitorStatus.running ? 'success' : 'info'" size="small">
              {{ monitorStatus.running ? '运行中' : '已停止' }}
            </el-tag>
          </div>
        </template>

        <el-form :model="monitorForm" label-width="100px" size="small">
          <el-form-item label="启用监控">
            <el-switch v-model="monitorForm.monitor_enabled" />
          </el-form-item>
          <el-form-item label="检查间隔">
            <el-input-number v-model="monitorForm.monitor_interval" :min="60" :max="86400" :step="60" style="width: 140px;" />
            <span style="margin-left: 8px;">秒</span>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="checkNow" :loading="checking">
              立即检查
            </el-button>
          </el-form-item>
        </el-form>

        <!-- 监控状态 -->
        <el-divider content-position="left">监控状态</el-divider>
        <div class="monitor-status">
          <div class="status-item">
            <span class="label">Token 状态:</span>
            <el-tag :type="monitorStatus.status?.token_valid ? 'success' : 'danger'" size="small">
              {{ monitorStatus.status?.token_valid ? '有效' : '失效' }}
            </el-tag>
          </div>
          <div class="status-item" v-if="monitorStatus.status?.last_check_at">
            <span class="label">上次检查:</span>
            <span>{{ formatTime(monitorStatus.status.last_check_at) }}</span>
          </div>
          <div class="status-item" v-if="monitorStatus.status?.last_error">
            <span class="label">最后错误:</span>
            <span class="error-text">{{ monitorStatus.status.last_error }}</span>
          </div>
        </div>
      </el-card>

      <!-- 通知设置 -->
      <el-card class="masonry-item">
        <template #header>
          <span class="card-header">通知设置</span>
        </template>

        <el-form :model="monitorForm" label-width="100px" size="small">
          <el-form-item label="启用通知">
            <el-switch v-model="monitorForm.notify_enabled" />
            <div class="form-tip">启用后，Token 失效时将发送通知</div>
          </el-form-item>
          <el-form-item label="默认渠道">
            <el-select v-model="monitorForm.default_notify_channel" placeholder="选择默认通知渠道" style="width: 200px;">
              <el-option label="Bark (iOS)" value="bark" />
              <el-option label="Server 酱 (微信)" value="serverchan" />
              <el-option label="Telegram" value="telegram" />
              <el-option label="PushPlus (微信)" value="pushplus" />
              <el-option label="钉钉机器人" value="dingtalk" />
              <el-option label="企业微信" value="wecom" />
            </el-select>
            <div class="form-tip">优先使用此渠道发送通知</div>
          </el-form-item>

          <!-- Bark (iOS) -->
          <template v-if="monitorForm.default_notify_channel === 'bark'">
            <el-divider content-position="left">Bark (iOS)</el-divider>
            <el-form-item label="Bark URL">
              <el-input v-model="monitorForm.bark_url" placeholder="https://api.day.app/xxxxx" />
            </el-form-item>
          </template>

          <!-- Server 酱 (微信) -->
          <template v-if="monitorForm.default_notify_channel === 'serverchan'">
            <el-divider content-position="left">Server 酱 (微信)</el-divider>
            <el-form-item label="SendKey">
              <el-input v-model="monitorForm.serverchan_key" placeholder="SCT..." show-password type="password" />
            </el-form-item>
          </template>

          <!-- Telegram -->
          <template v-if="monitorForm.default_notify_channel === 'telegram'">
            <el-divider content-position="left">Telegram</el-divider>
            <el-form-item label="Bot Token">
              <el-input v-model="monitorForm.telegram_bot_token" placeholder="123456:ABC..." show-password type="password" />
            </el-form-item>
            <el-form-item label="Chat ID">
              <el-input v-model="monitorForm.telegram_chat_id" placeholder="123456789" />
            </el-form-item>
          </template>

          <!-- PushPlus (微信) -->
          <template v-if="monitorForm.default_notify_channel === 'pushplus'">
            <el-divider content-position="left">PushPlus (微信)</el-divider>
            <el-form-item label="Token">
              <el-input v-model="monitorForm.pushplus_token" placeholder="Token" show-password type="password" />
            </el-form-item>
          </template>

          <!-- 钉钉机器人 -->
          <template v-if="monitorForm.default_notify_channel === 'dingtalk'">
            <el-divider content-position="left">钉钉机器人</el-divider>
            <el-form-item label="Webhook">
              <el-input v-model="monitorForm.dingtalk_webhook" placeholder="https://oapi.dingtalk.com/robot/send?access_token=..." />
            </el-form-item>
          </template>

          <!-- 企业微信 -->
          <template v-if="monitorForm.default_notify_channel === 'wecom'">
            <el-divider content-position="left">企业微信</el-divider>
            <el-form-item label="Webhook">
              <el-input v-model="monitorForm.wecom_webhook" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
            </el-form-item>
          </template>

          <el-form-item>
            <el-button type="primary" @click="testNotify" :loading="testingNotify">
              测试通知
            </el-button>
            <el-button type="success" @click="saveMonitorSettings" :loading="savingMonitor">
              保存设置
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </div>

    <!-- 通知记录（全宽） -->
    <el-card class="full-width-card">
      <template #header>
        <div class="card-header">
          <span>通知记录</span>
          <div>
            <el-button size="small" @click="loadNotifications">刷新</el-button>
            <el-button size="small" type="danger" @click="clearNotifications" :loading="clearingNotifications">
              清空记录
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="notifications" style="width: 100%" v-loading="loadingNotifications" size="small">
        <el-table-column prop="created_at" label="时间" width="160">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="event_type" label="渠道" width="120">
          <template #default="{ row }">
            <el-tag :type="getChannelTagType(row.event_type)" size="small">
              {{ getChannelLabel(row.event_type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="消息" />
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error_msg" label="错误信息" width="180">
          <template #default="{ row }">
            <span v-if="row.error_msg" class="error-text">{{ row.error_msg }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper" v-if="notificationTotal > 0">
        <el-pagination
          v-model:current-page="notificationPage"
          :page-size="10"
          :total="notificationTotal"
          layout="prev, pager, next"
          @current-change="loadNotifications"
          size="small"
        />
      </div>

      <el-empty v-if="!loadingNotifications && notifications.length === 0" description="暂无通知记录" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { settingsApi, monitorApi } from '../../api'

const settings = ref({
  initialized: false,
  refresh_token: '',
  access_token: '',
  root_folder_id: '',
  site_title: '',
  site_logo: ''
})

const cloudForm = ref({
  refresh_token: '',
  access_token: '',
  root_folder_id: ''
})

const siteForm = ref({
  site_title: '',
  site_logo: '',
  login_title: '',
  login_avatar: '',
  login_role_tag: '',
  login_level_tag: '',
  login_system_name: '',
  share_footer: ''
})

const passwordForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

// 监控相关
const monitorForm = ref({
  monitor_enabled: false,
  monitor_interval: 300,
  notify_enabled: false,
  default_notify_channel: 'bark',
  bark_url: '',
  serverchan_key: '',
  telegram_bot_token: '',
  telegram_chat_id: '',
  pushplus_token: '',
  dingtalk_webhook: '',
  wecom_webhook: ''
})

const monitorStatus = ref({
  running: false,
  status: null as any
})

const notifications = ref<any[]>([])
const notificationPage = ref(1)
const notificationTotal = ref(0)
const loadingNotifications = ref(false)
const testingNotify = ref(false)
const savingMonitor = ref(false)
const checking = ref(false)
const clearingNotifications = ref(false)

const saving = ref(false)
const testing = ref(false)
const updatingPwd = ref(false)

async function loadSettings() {
  try {
    const res = await settingsApi.get()
    settings.value = res.data
    cloudForm.value = {
      refresh_token: res.data.refresh_token,
      access_token: res.data.access_token || '',
      root_folder_id: res.data.root_folder_id
    }
    siteForm.value = {
      site_title: res.data.site_title,
      site_logo: res.data.site_logo,
      login_title: res.data.login_title || '',
      login_avatar: res.data.login_avatar || '',
      login_role_tag: res.data.login_role_tag || '',
      login_level_tag: res.data.login_level_tag || '',
      login_system_name: res.data.login_system_name || '',
      share_footer: res.data.share_footer || ''
    }
  } catch (error) {
    // 错误已处理
  }
}

async function testToken() {
  const refreshToken = cloudForm.value.refresh_token
  const accessToken = cloudForm.value.access_token
  const payload = {
    refresh_token: refreshToken && !refreshToken.includes('****') ? refreshToken : '',
    access_token: accessToken && !accessToken.includes('****') ? accessToken : ''
  }
  if (!payload.refresh_token && !payload.access_token) {
    ElMessage.warning('请输入完整的 Refresh Token 或 Access Token')
    return
  }
  testing.value = true
  try {
    await settingsApi.testToken(payload)
    ElMessage.success('Token 有效，连接成功！')
  } catch (error) {
    // 错误已处理
  } finally {
    testing.value = false
  }
}

async function saveCloudSettings() {
  saving.value = true
  try {
    await settingsApi.update({
      ...cloudForm.value,
      site_title: siteForm.value.site_title,
      site_logo: siteForm.value.site_logo
    })
    ElMessage.success('配置已保存，正在重新连接云盘...')
    setTimeout(() => loadSettings(), 2000)
  } catch (error) {
    // 错误已处理
  } finally {
    saving.value = false
  }
}

async function saveSiteSettings() {
  saving.value = true
  try {
    await settingsApi.update({
      refresh_token: '',
      access_token: '',
      root_folder_id: cloudForm.value.root_folder_id,
      ...siteForm.value
    })
    ElMessage.success('站点设置已保存')
  } catch (error) {
    // 错误已处理
  } finally {
    saving.value = false
  }
}

async function updatePassword() {
  if (passwordForm.value.new_password !== passwordForm.value.confirm_password) {
    ElMessage.error('两次输入的密码不一致')
    return
  }
  if (!passwordForm.value.old_password || !passwordForm.value.new_password) {
    ElMessage.warning('请填写完整')
    return
  }
  updatingPwd.value = true
  try {
    await settingsApi.updatePassword(
      passwordForm.value.old_password,
      passwordForm.value.new_password
    )
    ElMessage.success('密码已修改')
    passwordForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (error) {
    // 错误已处理
  } finally {
    updatingPwd.value = false
  }
}

// 监控相关函数
async function loadMonitorSettings() {
  try {
    const res = await monitorApi.getSettings()
    monitorForm.value = {
      monitor_enabled: res.data.monitor_enabled,
      monitor_interval: res.data.monitor_interval || 300,
      notify_enabled: res.data.notify_enabled,
      default_notify_channel: res.data.default_notify_channel || 'bark',
      bark_url: res.data.bark_url || '',
      serverchan_key: res.data.serverchan_key || '',
      telegram_bot_token: res.data.telegram_bot_token || '',
      telegram_chat_id: res.data.telegram_chat_id || '',
      pushplus_token: res.data.pushplus_token || '',
      dingtalk_webhook: res.data.dingtalk_webhook || '',
      wecom_webhook: res.data.wecom_webhook || ''
    }
  } catch (error) {
    // 错误已处理
  }
}

async function loadMonitorStatus() {
  try {
    const res = await monitorApi.getStatus()
    monitorStatus.value = {
      running: res.data.running,
      status: res.data.status
    }
  } catch (error) {
    // 错误已处理
  }
}

async function loadNotifications() {
  loadingNotifications.value = true
  try {
    const res = await monitorApi.getNotifications(notificationPage.value, 10)
    notifications.value = res.data.logs || []
    notificationTotal.value = res.data.total || 0
  } catch (error) {
    // 错误已处理
  } finally {
    loadingNotifications.value = false
  }
}

async function saveMonitorSettings() {
  savingMonitor.value = true
  try {
    // 过滤掉脱敏的值（包含****）
    const filterMasked = (v: string) => v.includes('****') ? '' : v
    await monitorApi.updateSettings({
      monitor_enabled: monitorForm.value.monitor_enabled,
      monitor_interval: monitorForm.value.monitor_interval,
      notify_enabled: monitorForm.value.notify_enabled,
      bark_url: filterMasked(monitorForm.value.bark_url),
      serverchan_key: filterMasked(monitorForm.value.serverchan_key),
      telegram_bot_token: filterMasked(monitorForm.value.telegram_bot_token),
      telegram_chat_id: monitorForm.value.telegram_chat_id,
      pushplus_token: filterMasked(monitorForm.value.pushplus_token),
      dingtalk_webhook: filterMasked(monitorForm.value.dingtalk_webhook),
      wecom_webhook: filterMasked(monitorForm.value.wecom_webhook)
    })
    ElMessage.success('监控设置已保存')
    loadMonitorStatus()
  } catch (error) {
    // 错误已处理
  } finally {
    savingMonitor.value = false
  }
}

async function testNotify() {
  testingNotify.value = true
  try {
    const res = await monitorApi.testNotify()
    const results = res.data as Record<string, string>
    const messages = Object.entries(results).map(([k, v]) => `${getChannelLabel(k)}: ${v}`).join('\n')
    ElMessage.success({ message: messages || '测试完成', duration: 5000 })
    loadNotifications()
  } catch (error) {
    // 错误已处理
  } finally {
    testingNotify.value = false
  }
}

async function checkNow() {
  checking.value = true
  try {
    await monitorApi.checkNow()
    ElMessage.success('检查完成')
    loadMonitorStatus()
    loadNotifications()
  } catch (error) {
    // 错误已处理
  } finally {
    checking.value = false
  }
}

async function clearNotifications() {
  try {
    await ElMessageBox.confirm('确定要清空所有通知记录吗？', '确认', {
      type: 'warning'
    })
    clearingNotifications.value = true
    await monitorApi.clearNotifications()
    ElMessage.success('通知记录已清空')
    loadNotifications()
  } catch (error) {
    if ((error as any) !== 'cancel') {
      // 错误已处理
    }
  } finally {
    clearingNotifications.value = false
  }
}

function formatTime(timeStr: string) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

function getChannelTagType(channel: string) {
  const types: Record<string, string> = {
    'bark': 'success',
    'serverchan': 'primary',
    'telegram': 'info',
    'pushplus': 'warning',
    'dingtalk': 'primary',
    'wecom': 'success',
    'token_invalid': 'danger',
    'token_refreshed': 'success'
  }
  return types[channel] || 'info'
}

function getChannelLabel(channel: string) {
  const labels: Record<string, string> = {
    'bark': 'Bark',
    'serverchan': 'Server酱',
    'telegram': 'Telegram',
    'pushplus': 'PushPlus',
    'dingtalk': '钉钉',
    'wecom': '企业微信',
    'token_invalid': 'Token失效',
    'token_refreshed': 'Token刷新'
  }
  return labels[channel] || channel
}

onMounted(() => {
  loadSettings()
  loadMonitorSettings()
  loadMonitorStatus()
  loadNotifications()
})
</script>

<style scoped>
.settings-page {
  max-width: 1400px;
}

/* 瀑布流布局 */
.masonry-container {
  column-count: 2;
  column-gap: 16px;
}

.masonry-item {
  break-inside: avoid;
  margin-bottom: 16px;
}

/* 全宽卡片 */
.full-width-card {
  margin-top: 16px;
}

.card-header {
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.form-tip a {
  color: #409eff;
}

.monitor-status {
  padding: 0 10px;
}

.status-item {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
  font-size: 13px;
}

.status-item .label {
  min-width: 70px;
  color: #606266;
}

.error-text {
  color: #f56c6c;
  font-size: 12px;
}

.pagination-wrapper {
  margin-top: 12px;
  display: flex;
  justify-content: center;
}

/* 响应式：小屏单列 */
@media (max-width: 900px) {
  .masonry-container {
    column-count: 1;
  }
}

/* 移动端优化 */
@media (max-width: 768px) {
  .masonry-container {
    column-count: 1;
  }

  /* 按钮优化 */
  :deep(.el-button) {
    box-shadow: 3px 3px 0 #000 !important;
    padding: 10px 16px !important;
    font-size: 12px !important;
  }

  :deep(.el-button:hover) {
    box-shadow: 5px 5px 0 #000 !important;
  }

  :deep(.el-button--small) {
    padding: 8px 12px !important;
    font-size: 11px !important;
    box-shadow: 3px 3px 0 #000 !important;
  }

  /* 表单项优化 */
  :deep(.el-form-item) {
    flex-direction: column;
    align-items: flex-start;
  }

  :deep(.el-form-item__label) {
    width: auto !important;
    margin-bottom: 5px;
    padding-right: 0 !important;
  }

  :deep(.el-form-item__content) {
    width: 100%;
    margin-left: 0 !important;
  }

  /* 表格优化 */
  :deep(.el-table) {
    font-size: 12px;
  }

  :deep(.el-table .el-table__cell) {
    padding: 8px 4px;
  }

  /* 隐藏错误信息列（第5列） */
  :deep(.el-table th:nth-child(5)),
  :deep(.el-table td:nth-child(5)) {
    display: none;
  }

  /* 输入框优化 */
  :deep(.el-input__wrapper) {
    box-shadow: 3px 3px 0 #ccc !important;
  }

  :deep(.el-select .el-input__wrapper) {
    box-shadow: 3px 3px 0 #000 !important;
  }

  /* 卡片优化 */
  :deep(.el-card) {
    box-shadow: 3px 3px 0 #000;
  }
}

/* ========================================
   新粗犷主义按钮样式 - NEO BRUTALISM
   ======================================== */
:deep(.el-button) {
  border: 4px solid #000 !important;
  border-radius: 0 !important;
  font-weight: 900 !important;
  text-transform: uppercase !important;
  letter-spacing: 1px !important;
  transition: all 0.08s ease-out !important;
  position: relative !important;
  font-size: 14px !important;
  padding: 12px 24px !important;
}

/* Primary 按钮 - 纯黑 */
:deep(.el-button--primary) {
  background: #000 !important;
  color: #fff !important;
  box-shadow: 6px 6px 0 #000 !important;
  border-color: #000 !important;
}

:deep(.el-button--primary:hover) {
  background: #222 !important;
  transform: translate(-3px, -3px) !important;
  box-shadow: 9px 9px 0 #000 !important;
}

:deep(.el-button--primary:active) {
  transform: translate(2px, 2px) !important;
  box-shadow: 4px 4px 0 #000 !important;
}

/* Success 按钮 - 荧光黄绿 */
:deep(.el-button--success) {
  background: #ccff00 !important;
  color: #000 !important;
  box-shadow: 6px 6px 0 #000 !important;
  border-color: #000 !important;
}

:deep(.el-button--success:hover) {
  background: #e6ff4d !important;
  transform: translate(-3px, -3px) !important;
  box-shadow: 9px 9px 0 #000 !important;
}

:deep(.el-button--success:active) {
  transform: translate(2px, 2px) !important;
  box-shadow: 4px 4px 0 #000 !important;
}

/* Danger 按钮 - 亮红 */
:deep(.el-button--danger) {
  background: #ff2d55 !important;
  color: #fff !important;
  box-shadow: 6px 6px 0 #000 !important;
  border-color: #000 !important;
}

:deep(.el-button--danger:hover) {
  background: #ff4d6d !important;
  transform: translate(-3px, -3px) !important;
  box-shadow: 9px 9px 0 #000 !important;
}

:deep(.el-button--danger:active) {
  transform: translate(2px, 2px) !important;
  box-shadow: 4px 4px 0 #000 !important;
}

/* Default 按钮 - 白底黑框 */
:deep(.el-button--default) {
  background: #fff !important;
  color: #000 !important;
  box-shadow: 6px 6px 0 #000 !important;
  border-color: #000 !important;
}

:deep(.el-button--default:hover) {
  background: #f0f0f0 !important;
  transform: translate(-3px, -3px) !important;
  box-shadow: 9px 9px 0 #000 !important;
}

:deep(.el-button--default:active) {
  transform: translate(2px, 2px) !important;
  box-shadow: 4px 4px 0 #000 !important;
}

/* Small 尺寸按钮调整 */
:deep(.el-button--small) {
  padding: 8px 16px !important;
  font-size: 12px !important;
  border-width: 3px !important;
  box-shadow: 5px 5px 0 #000 !important;
}

:deep(.el-button--small:hover) {
  box-shadow: 7px 7px 0 #000 !important;
}

/* ========================================
   新粗犷主义下拉选择样式
   ======================================== */
:deep(.el-select .el-input__wrapper) {
  border: 4px solid #000 !important;
  border-radius: 0 !important;
  box-shadow: 5px 5px 0 #000 !important;
  transition: all 0.08s ease-out !important;
  background: #fff !important;
}

:deep(.el-select .el-input__wrapper:hover) {
  transform: translate(-2px, -2px);
  box-shadow: 7px 7px 0 #000 !important;
}

:deep(.el-select .el-input.is-focus .el-input__wrapper) {
  border-color: #ccff00 !important;
  box-shadow: 5px 5px 0 #ccff00 !important;
}

/* ========================================
   新粗犷主义输入框样式
   ======================================== */
:deep(.el-input__wrapper) {
  border: 3px solid #000 !important;
  border-radius: 0 !important;
  box-shadow: 4px 4px 0 #ccc !important;
  transition: all 0.08s ease-out !important;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 5px 5px 0 #999 !important;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #ccff00 !important;
  box-shadow: 4px 4px 0 #ccff00 !important;
}

/* ========================================
   新粗犷主义开关样式
   ======================================== */
:deep(.el-switch) {
  --el-switch-on-color: #ccff00 !important;
  --el-switch-off-color: #ccc !important;
}

:deep(.el-switch__core) {
  border: 3px solid #000 !important;
  border-radius: 0 !important;
}

/* ========================================
   新粗犷主义分割线样式
   ======================================== */
:deep(.el-divider) {
  border-color: #000 !important;
  border-width: 2px !important;
}

:deep(.el-divider__text) {
  background: #fff !important;
  font-weight: 900 !important;
  text-transform: uppercase !important;
  letter-spacing: 1px !important;
  color: #000 !important;
}
</style>
