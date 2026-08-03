<template>
  <div class="shares-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>分享列表</span>
          <el-button type="primary" size="small" :icon="Refresh" @click="loadShares">
            刷新
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="shares" class="shares-table" table-layout="fixed" style="width: 100%">
        <el-table-column label="名称" min-width="275" class-name="name-column">
          <template #default="{ row }">
            <div class="share-name">
              <el-icon :size="18" :color="row.target_type === 'folder' ? '#409eff' : '#909399'">
                <Folder v-if="row.target_type === 'folder'" />
                <Document v-else />
              </el-icon>
              <span class="share-name-text" :title="row.target_name">{{ row.target_name }}</span>
              <span class="type-badge" :class="isDirectFile(row) ? 'direct' : 'share'">
                {{ isDirectFile(row) ? '直连' : row.target_type === 'folder' ? '目录分享' : '分享' }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="短链接" width="95" class-name="link-column">
          <template #default="{ row }">
            <div class="link-actions">
              <el-tooltip content="复制链接" placement="top" :hide-after="0">
                <el-button
                  class="btn-brutal-icon"
                  @click="copyLink(row.share_code, isDirectFile(row))"
                >
                  <el-icon><CopyDocument /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="打开链接" placement="top" :hide-after="0">
                <el-button
                  class="btn-brutal-icon secondary"
                  @click="openLink(row.share_code, isDirectFile(row))"
                >
                  <el-icon><TopRight /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="密码" width="65" class-name="password-column">
          <template #default="{ row }">
            <span v-if="row.password" class="status-tag yes">有</span>
            <span v-else class="status-tag no">-</span>
          </template>
        </el-table-column>
        <el-table-column label="过期时间" width="110" class-name="expire-column">
          <template #default="{ row }">
             {{ row.expire_at ? formatDate(row.expire_at) : '永不过期' }}
          </template>
        </el-table-column>
        <el-table-column label="访问/下载" width="95" class-name="stats-column">
          <template #default="{ row }">
            <span style="font-family: monospace; font-weight: bold;">
              <template v-if="isDirectFile(row)">引用 {{ row.download_count }}</template>
              <template v-else>{{ row.view_count }} / {{ row.download_count }}</template>
              <span v-if="row.max_downloads > 0" style="color: #909399;">
                (限{{ row.max_downloads }}次)
              </span>
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right" class-name="actions-column">
          <template #default="{ row }">
            <div class="row-actions">
               <el-tooltip v-if="isDirectFile(row)" content="嵌入代码" placement="top" :hide-after="0">
                <el-button class="btn-brutal-icon" @click="showEmbed(row)">
                  <el-icon><Connection /></el-icon>
                </el-button>
              </el-tooltip>
               <el-button class="btn-brutal-icon" @click="showQRCode(row)">
                <el-icon><Grid /></el-icon>
              </el-button>
              <el-button class="btn-brutal-small" @click="openEditDialog(row)">
                编辑
              </el-button>
              <el-popconfirm
                title="确定删除此分享？"
                @confirm="deleteShare(row.id)"
              >
                <template #reference>
                  <el-button class="btn-brutal-icon danger">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </template>
              </el-popconfirm>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadShares"
        />
      </div>
    </el-card>

    <!-- 编辑对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑分享" width="460px">
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="短链接">
          <el-input v-model="editForm.share_code">
            <template #prepend>/s/</template>
          </el-input>
        </el-form-item>
        <el-form-item v-if="editForm.target_type === 'file'" label="直连模式">
          <el-switch v-model="editForm.is_direct" />
          <div class="form-tip">图床/外链嵌入，生成 /f/ 链接，不支持密码</div>
        </el-form-item>
        <el-form-item v-else label="分享模式">
          <div class="form-tip">文件夹固定使用 /s/ 目录树分享</div>
        </el-form-item>
        <el-form-item label="访问密码">
          <el-input
            v-model="editForm.password"
            placeholder="留空则无需密码"
            show-password
            :disabled="editForm.is_direct"
          />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="editForm.expire_at"
            type="datetime"
            placeholder="选择过期时间"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="下载次数">
          <el-input-number
            v-model="editForm.max_downloads"
            :min="0"
            :max="9999"
            placeholder="0表示不限制"
            style="width: 100%"
          />
          <div class="form-tip">设置为 0 表示不限制下载次数</div>
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="editForm.description"
            type="textarea"
            :rows="2"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="editForm.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="updating" @click="updateShare">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 嵌入代码对话框 -->
    <el-dialog v-model="embedDialogVisible" title="嵌入代码" width="520px">
      <div class="embed-list">
        <div v-for="item in embedSnippets" :key="item.label" class="embed-item">
          <div class="embed-label">{{ item.label }}</div>
          <div class="embed-row">
            <el-input v-model="item.code" readonly />
            <el-button class="btn-brutal-small" @click="copyText(item.code)">复制</el-button>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 二维码对话框 -->
    <el-dialog v-model="qrcodeDialogVisible" title="分享二维码" width="360px">
      <div class="qrcode-content" v-loading="loadingQRCode">
        <img v-if="qrcodeData" :src="qrcodeData" alt="QR Code" class="qrcode-image" />
        <div class="qrcode-url">{{ qrcodeUrl }}</div>
        <div class="qrcode-actions">
          <el-button type="primary" @click="copyLink(currentShareCode)">
            <el-icon><CopyDocument /></el-icon> 复制链接
          </el-button>
          <el-button @click="downloadQRCode">
            <el-icon><Download /></el-icon> 下载二维码
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Download, Delete, TopRight } from '@element-plus/icons-vue'
import { shareApi } from '../../api'

interface Share {
  id: number
  share_code: string
  target_type: string
  target_id: string
  target_name: string
  password: string
  expire_at: string | null
  description: string
  is_active: boolean
  view_count: number
  download_count: number
  max_downloads: number
  is_direct: boolean
  created_at: string
}

const loading = ref(false)
const shares = ref<Share[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const editDialogVisible = ref(false)
const updating = ref(false)
const editForm = ref({
  id: 0,
  target_type: 'file',
  share_code: '',
  password: '',
  expire_at: null as Date | null,
  description: '',
  is_active: true,
  max_downloads: 0,
  is_direct: false
})

// QR Code
const qrcodeDialogVisible = ref(false)
const loadingQRCode = ref(false)
const qrcodeData = ref('')
const qrcodeUrl = ref('')
const currentShareCode = ref('')

async function loadShares() {
  loading.value = true
  try {
    const res = await shareApi.list(page.value, pageSize.value)
    shares.value = res.data.shares || []
    total.value = res.data.total
  } catch (error) {
    // 错误已处理
  } finally {
    loading.value = false
  }
}

function getShareUrl(code: string, isDirect = false): string {
  return `${window.location.origin}/${isDirect ? 'f' : 's'}/${code}`
}

function isDirectFile(share: Share): boolean {
  return share.target_type === 'file' && share.is_direct
}

function openLink(code: string, isDirect = false) {
  window.open(getShareUrl(code, isDirect), '_blank')
}

function copyLink(code: string, isDirect = false) {
  const url = getShareUrl(code, isDirect)
  navigator.clipboard.writeText(url)
  ElMessage.success('链接已复制')
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制')
}

// 嵌入代码（仅直连分享）
const embedDialogVisible = ref(false)
const embedSnippets = ref<{ label: string; code: string }[]>([])

function showEmbed(share: Share) {
  const url = getShareUrl(share.share_code, true)
  const name = share.target_name || 'file'
  embedSnippets.value = [
    { label: '直链 URL', code: url },
    // referrerpolicy=no-referrer：联通 CDN 只放行空 Referer，跟随 302 时不带来源才能出图。
    // video/audio 不支持该属性（whatwg/html#7822），视频外链需嵌入页自设 no-referrer 策略。
    { label: 'HTML 图片', code: `<img src="${url}" alt="${name}" referrerpolicy="no-referrer">` },
    { label: 'Markdown', code: `![${name}](${url})` },
    { label: 'HTML 视频', code: `<video src="${url}" controls></video>` },
    { label: 'BBCode', code: `[img]${url}[/img]` }
  ]
  embedDialogVisible.value = true
}

function openEditDialog(share: Share) {
  editForm.value = {
    id: share.id,
    target_type: share.target_type,
    share_code: share.share_code,
    password: share.password,
    expire_at: share.expire_at ? new Date(share.expire_at) : null,
    description: share.description,
    is_active: share.is_active,
    max_downloads: share.max_downloads || 0,
    is_direct: isDirectFile(share)
  }
  editDialogVisible.value = true
}

async function updateShare() {
  updating.value = true
  try {
    const data: any = {
      share_code: editForm.value.share_code,
      password: editForm.value.is_direct ? '' : editForm.value.password,
      description: editForm.value.description,
      is_active: editForm.value.is_active,
      max_downloads: editForm.value.max_downloads,
      is_direct: editForm.value.target_type === 'file' && editForm.value.is_direct
    }
    if (editForm.value.expire_at) {
      data.expire_at = editForm.value.expire_at.toISOString()
    }
    await shareApi.update(editForm.value.id, data)
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    loadShares()
  } catch (error) {
    // 错误已处理
  } finally {
    updating.value = false
  }
}

async function deleteShare(id: number) {
  try {
    await shareApi.delete(id)
    ElMessage.success('删除成功')
    loadShares()
  } catch (error) {
    // 错误已处理
  }
}

async function showQRCode(share: Share) {
  currentShareCode.value = share.share_code
  qrcodeDialogVisible.value = true
  loadingQRCode.value = true
  qrcodeData.value = ''
  qrcodeUrl.value = ''

  try {
    const res = await shareApi.getQRCode(share.id)
    qrcodeData.value = res.data.qrcode
    qrcodeUrl.value = res.data.share_url
  } catch (error) {
    // 错误已处理
  } finally {
    loadingQRCode.value = false
  }
}

function downloadQRCode() {
  if (!qrcodeData.value) return
  const link = document.createElement('a')
  link.href = qrcodeData.value
  link.download = `share-${currentShareCode.value}.png`
  link.click()
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  loadShares()
})
</script>

<style scoped>
.shares-page {
  max-width: 1400px;
}

.embed-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.embed-label {
  font-weight: 700;
  font-size: 13px;
  margin-bottom: 4px;
}

.embed-row {
  display: flex;
  gap: 8px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
}

.share-name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.share-name-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.shares-table :deep(.el-table__cell) {
  padding: 10px 8px !important;
}

.shares-table :deep(.name-column .cell),
.shares-table :deep(.expire-column .cell),
.shares-table :deep(.el-table__body-wrapper .stats-column .cell) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.shares-table :deep(th .cell) {
  overflow: visible;
  padding: 0 4px;
  text-overflow: clip;
  white-space: nowrap;
}

.shares-table :deep(.link-column .cell),
.shares-table :deep(.password-column .cell),
.shares-table :deep(.stats-column .cell),
.shares-table :deep(.actions-column .cell) {
  text-align: center;
}

.type-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 1px 6px;
  border: 1px solid black;
  border-radius: 2px;
}
.type-badge.direct {
  background: var(--acid-green, #ccff00);
  color: black;
}
.type-badge.share {
  background: #f0f0f0;
  color: #666;
}

.text-muted {
  color: #909399;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.qrcode-content {
  text-align: center;
  padding: 20px;
}

.qrcode-image {
  width: 200px;
  height: 200px;
  margin-bottom: 16px;
}

.qrcode-url {
  font-size: 12px;
  color: #606266;
  margin-bottom: 20px;
  word-break: break-all;
}

.qrcode-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 5px;
}
</style>
