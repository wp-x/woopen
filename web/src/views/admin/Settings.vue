<template>
  <div class="settings-page">
    <el-row :gutter="20">
      <!-- 云盘配置 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>云盘配置</span>
              <el-tag :type="settings.initialized ? 'success' : 'warning'" size="small">
                {{ settings.initialized ? '已连接' : '未连接' }}
              </el-tag>
            </div>
          </template>

          <el-form :model="cloudForm" label-width="120px">
            <el-form-item label="Refresh Token">
              <el-input
                v-model="cloudForm.refresh_token"
                type="password"
                show-password
                placeholder="从浏览器开发者工具获取"
              />
              <div class="form-tip">
                登录 <a href="https://pan.wo.cn" target="_blank">联通云盘</a> 后，
                在开发者工具 Network 中找到 refresh_token
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
                <strong>可选：</strong>如果只拿到一个 Token，填写在 Refresh Token，Access Token 留空即可。
                若两个 Token 都有，建议同时填写。
              </div>
            </el-form-item>
            <el-form-item label="根目录 ID">
              <el-input
                v-model="cloudForm.root_folder_id"
                placeholder="留空使用根目录"
              />
              <div class="form-tip">限制只显示指定文件夹的内容</div>
            </el-form-item>
            <el-form-item>
              <el-button class="btn-brutal-small" @click="testToken" :loading="testing" style="background: black !important; color: white !important;">
                测试连接
              </el-button>
              <el-button class="btn-brutal-small" @click="saveCloudSettings" :loading="saving" style="background: var(--acid-green) !important;">
                保存配置
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 站点设置 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <span class="card-header">站点设置</span>
          </template>

          <el-form :model="siteForm" label-width="120px">
            <el-form-item label="站点标题">
              <el-input v-model="siteForm.site_title" placeholder="WoOpen" />
            </el-form-item>
            <el-form-item label="站点 Logo">
              <el-input v-model="siteForm.site_logo" placeholder="Logo URL（可选）" />
            </el-form-item>
            
            <el-divider content-position="left">登录界面自定义</el-divider>
            
            <el-form-item label="窗口标题">
              <el-input v-model="siteForm.login_title" placeholder="WoOpen_Auth_v10.exe" />
              <div class="form-tip">登录窗口顶部的标题栏文字</div>
            </el-form-item>
            <el-form-item label="头像图标">
              <el-input v-model="siteForm.login_avatar" placeholder="💀 或图片URL" />
              <div class="form-tip">支持 emoji 或图片 URL，如：💀、🔐、或 https://example.com/avatar.png</div>
            </el-form-item>
            <el-form-item label="角色标签">
              <el-input v-model="siteForm.login_role_tag" placeholder="ROLE: ADMIN" />
            </el-form-item>
            <el-form-item label="等级标签">
              <el-input v-model="siteForm.login_level_tag" placeholder="LEVEL: 99" />
            </el-form-item>
            <el-form-item label="系统名称">
              <el-input v-model="siteForm.login_system_name" placeholder="WOOPEN CLOUD SYSTEM" />
              <div class="form-tip">登录页左侧显示的大标题（支持换行，用空格分隔）</div>
            </el-form-item>
            
            <el-divider content-position="left">分享界面自定义</el-divider>
            
            <el-form-item label="底部署名">
              <el-input v-model="siteForm.share_footer" placeholder="Powered by WOOPEN_OS // BRUTAL_EDITION" />
              <div class="form-tip">分享页面底部的版权/署名信息</div>
            </el-form-item>
            
            <el-form-item>
              <el-button class="btn-brutal-small" @click="saveSiteSettings" :loading="saving" style="background: var(--acid-green) !important;">
                保存站点设置
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 修改密码 -->
    <el-row :gutter="20" class="mt-20">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span class="card-header">修改密码</span>
          </template>

          <el-form :model="passwordForm" label-width="100px">
            <el-form-item label="原密码">
              <el-input
                v-model="passwordForm.old_password"
                type="password"
                show-password
              />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input
                v-model="passwordForm.new_password"
                type="password"
                show-password
              />
            </el-form-item>
            <el-form-item label="确认密码">
              <el-input
                v-model="passwordForm.confirm_password"
                type="password"
                show-password
              />
            </el-form-item>
            <el-form-item>
              <el-button class="btn-brutal-small" @click="updatePassword" :loading="updatingPwd" style="background: var(--acid-pink) !important; color: white !important;">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { settingsApi } from '../../api'

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
    // 延迟重载以等待后端重新初始化
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
      refresh_token: '',  // 不更新token
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

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-page {
  max-width: 1200px;
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
}

.form-tip a {
  color: #409eff;
}

.mt-20 {
  margin-top: 20px;
}
</style>
