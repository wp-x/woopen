<template>
  <div class="files-page">
    <!-- 面包屑导航 -->
    <el-card class="breadcrumb-card">
      <div class="breadcrumb-row">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item>
            <el-button type="primary" text size="small" @click="navigateTo('')">
              <el-icon><HomeFilled /></el-icon> 根目录
            </el-button>
          </el-breadcrumb-item>
          <el-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="index">
            <el-button type="primary" text size="small" @click="navigateTo(item.id)">
              {{ item.name }}
            </el-button>
          </el-breadcrumb-item>
        </el-breadcrumb>
        <div class="breadcrumb-actions" v-if="selectedFiles.length > 0">
          <el-tag type="info" size="small">已选择 {{ selectedFiles.length }} 项</el-tag>
          <el-button type="primary" size="small" @click="openBatchShareDialog">
            <el-icon><Share /></el-icon> 批量分享
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 文件列表 -->
    <el-card class="files-card">
      <template #header>
        <div class="card-header">
          <div class="card-header-left">
            <span>文件列表</span>
          </div>
          <div class="card-header-right">
            <!-- 排序 -->
            <el-dropdown @command="handleSort">
              <el-button size="small">
                <el-icon><Sort /></el-icon>
                {{ sortLabels[sortBy] }}
                <el-icon :class="{ 'rotate-180': sortOrder === 'desc' }"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="name">按名称</el-dropdown-item>
                  <el-dropdown-item command="size">按大小</el-dropdown-item>
                  <el-dropdown-item command="time">按时间</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <!-- 视图切换 -->
            <el-button-group>
              <el-button
                :type="viewMode === 'list' ? 'primary' : 'default'"
                size="small"
                @click="viewMode = 'list'"
              >
                <el-icon><List /></el-icon>
              </el-button>
              <el-button
                :type="viewMode === 'grid' ? 'primary' : 'default'"
                size="small"
                @click="viewMode = 'grid'"
              >
                <el-icon><Grid /></el-icon>
              </el-button>
            </el-button-group>
            <el-button type="primary" size="small" :icon="Refresh" @click="loadFiles">
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 列表视图 -->
      <el-table
        v-if="viewMode === 'list'"
        v-loading="loading"
        :data="filteredFiles"
        style="width: 100%"
        @row-dblclick="handleRowClick"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column label="名称" min-width="300">
          <template #default="{ row }">
            <div class="file-name" @click="handleRowClick(row)">
              <el-icon :size="24" color="var(--pure-black)" class="file-icon-brutal">
                <Folder v-if="row.is_dir" />
                <component v-else :is="getExtIcon(row.name)" />
              </el-icon>
              <span class="name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            {{ row.is_dir ? '-' : formatSize(row.size) }}
          </template>
        </el-table-column>
        <el-table-column label="修改时间" width="200">
          <template #default="{ row }">
            {{ formatDate(row.mod_time) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <div class="row-actions">
              <el-button
                class="btn-brutal-icon"
                @click.stop="openShareDialog(row)"
              >
                <el-icon><Share /></el-icon>
              </el-button>
              <el-button
                v-if="!row.is_dir && canPreview(row.name)"
                class="btn-brutal-icon"
                style="background: var(--acid-green) !important;"
                @click.stop="openPreview(row)"
              >
                <el-icon><View /></el-icon>
              </el-button>
              <el-button
                v-if="!row.is_dir"
                class="btn-brutal-icon"
                @click.stop="downloadFile(row)"
              >
                <el-icon><Download /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 网格视图 -->
      <div v-else-if="viewMode === 'grid'" v-loading="loading" class="grid-view">
        <div
          v-for="file in filteredFiles"
          :key="file.id"
          class="grid-item"
          :class="{ selected: selectedFiles.includes(file) }"
          @click="toggleSelect(file)"
          @dblclick="handleRowClick(file)"
        >
          <div class="grid-checkbox">
            <el-checkbox
              :model-value="selectedFiles.includes(file)"
              @click.stop
              @change="toggleSelect(file)"
            />
          </div>
          <div class="grid-icon">
            <el-icon :size="56" color="var(--pure-black)" class="file-icon-brutal">
              <Folder v-if="file.is_dir" />
              <component v-else :is="getExtIcon(file.name)" />
            </el-icon>
          </div>
          <div class="grid-name">{{ file.name }}</div>
          <div class="grid-size" v-if="!file.is_dir">{{ formatSize(file.size) }}</div>
          <div class="grid-actions">
            <el-button
              class="btn-brutal-icon"
              @click.stop="openShareDialog(file)"
            >
              <el-icon><Share /></el-icon>
            </el-button>
            <el-button
              v-if="!file.is_dir && canPreview(file.name)"
              class="btn-brutal-icon"
              style="background: var(--acid-green) !important;"
              @click.stop="openPreview(file)"
            >
              <el-icon><View /></el-icon>
            </el-button>
            <el-button
              v-if="!file.is_dir"
              class="btn-brutal-icon"
              @click.stop="downloadFile(file)"
            >
              <el-icon><Download /></el-icon>
            </el-button>
          </div>
        </div>
        <el-empty v-if="filteredFiles.length === 0" description="暂无文件" />
      </div>

      <div v-if="files.length === 0 && !loading && viewMode === 'list'" class="empty-state">
        <el-empty description="暂无文件" />
      </div>
    </el-card>

    <!-- 创建分享对话框 (Neo-Brutalist) -->
    <div v-if="shareDialogVisible" class="preview-overlay" @click.self="shareDialogVisible = false">
      <div class="window-frame active" style="width: 500px; max-width: 95vw;">
        <div class="window-header">
           <span>创建分享 // CONFIG</span>
           <div class="window-controls"><button @click="shareDialogVisible = false">X</button></div>
        </div>
        
        <div class="brutal-form-content">
           <div class="form-group">
              <label class="brutal-label-tag">文件名_TARGET</label>
              <div class="static-value-box">{{ shareForm.target_name }}</div>
           </div>

           <div class="form-group">
              <label class="brutal-label-tag">自定义链接_LINK</label>
              <div class="input-group-brutal">
                 <span class="prefix">/s/</span>
                 <input v-model="shareForm.share_code" class="brutal-input-reset" placeholder="留空自动生成 (Auto Gen)">
              </div>
           </div>

           <div class="form-group">
              <label class="brutal-label-tag">访问密码_PASS</label>
              <input v-model="shareForm.password" class="brutal-input-reset" type="password" placeholder="留空则公开 (Public)">
           </div>

           <div class="form-group">
              <label class="brutal-label-tag">过期时间_EXPIRE</label>
              <div class="date-picker-wrapper-brutal">
                 <el-date-picker 
                   v-model="shareForm.expire_at" 
                   type="datetime" 
                   placeholder="选择过期时间 (Optional)" 
                   style="width: 100% !important; height: 42px;"
                 />
              </div>
           </div>
           
           <div class="form-group">
              <label class="brutal-label-tag">备注信息_NOTE</label>
              <textarea v-model="shareForm.description" class="brutal-input-reset" rows="2" placeholder="仅管理员可见"></textarea>
           </div>
           
           <div class="form-actions-brutal">
              <button class="btn-brutal btn-cancel" @click="shareDialogVisible = false">取消</button>
              <button class="btn-brutal btn-confirm" @click="createShare">
                 {{ creating ? '创建中...' : '立即创建' }}
              </button>
           </div>
        </div>
      </div>
    </div>

    <!-- 批量分享对话框 (Neo-Brutalist) -->
    <div v-if="batchShareDialogVisible" class="preview-overlay" @click.self="batchShareDialogVisible = false">
      <div class="window-frame active" style="width: 500px; max-width: 95vw;">
        <div class="window-header">
           <span>批量分享 // BATCH ({{ selectedFiles.length }})</span>
           <div class="window-controls"><button @click="batchShareDialogVisible = false">X</button></div>
        </div>
        
        <div class="brutal-form-content">
           <div class="batch-info-box">
              <span style="font-weight: bold;">[INFO]</span> 即将为 <span style="background: black; color: white; padding: 0 5px;">{{ selectedFiles.length }}</span> 个文件生成独立链接
           </div>

           <div class="batch-file-list-brutal">
              <div v-for="file in selectedFiles" :key="file.id" class="batch-file-item-brutal">
                <span>{{ file.name }}</span>
              </div>
           </div>

           <div class="form-group" style="margin-top: 20px;">
              <label class="brutal-label-tag">统一密码_ALL</label>
              <input v-model="batchShareForm.password" class="brutal-input-reset" type="password" placeholder="留空则公开 (Public)">
           </div>

           <div class="form-group">
              <label class="brutal-label-tag">统一过期_EXPIRE</label>
              <div class="date-picker-wrapper-brutal">
                 <el-date-picker 
                   v-model="batchShareForm.expire_at" 
                   type="datetime" 
                   placeholder="选择过期时间 (Optional)" 
                   style="width: 100% !important; height: 42px;"
                 />
              </div>
           </div>
           
           <div class="form-actions-brutal">
              <button class="btn-brutal btn-cancel" @click="batchShareDialogVisible = false">取消</button>
              <button class="btn-brutal btn-confirm" @click="createBatchShare">
                 {{ batchCreating ? '处理中...' : '确认生成' }}
              </button>
           </div>
        </div>
      </div>
    </div>

    <!-- 批量分享结果对话框 (Neo-Brutalist) -->
    <div v-if="batchResultDialogVisible" class="preview-overlay" @click.self="batchResultDialogVisible = false">
      <div class="window-frame active" style="width: 650px; max-width: 95vw;">
        <div class="window-header">
           <span>批量任务报告 // REPORT</span>
           <div class="window-controls"><button @click="batchResultDialogVisible = false">X</button></div>
        </div>
        
        <div class="share-receipt">
           <!-- 成功横幅 -->
           <div class="success-banner-box" style="font-size: 18px; padding: 10px;">
              <div class="star-icon">★</div>
              <span>成功创建 {{ batchResults.length }} 个链接</span>
              <div class="star-icon">★</div>
           </div>

           <!-- 结果列表 -->
           <div class="batch-result-list">
              <div class="batch-result-header">
                 <span style="flex:1">文件名</span>
                 <span style="flex:2">生成链接</span>
                 <span style="width: 50px">操作</span>
              </div>
              <div class="batch-result-scroll">
                <div v-for="res in batchResults" :key="res.share_code" class="batch-result-row">
                   <div class="res-name">{{ res.target_name }}</div>
                   <div class="res-link">/s/{{ res.share_code }}</div>
                   <button class="btn-icon-tiny" @click="copyLink(res.share_code)">
                     <el-icon><CopyDocument /></el-icon>
                   </button>
                </div>
              </div>
           </div>

           <!-- 操作按钮组 -->
           <div class="action-buttons-row" style="margin-top: 20px;">
              <button class="btn-brutal btn-copy" @click="copyAllLinks">
                 <el-icon><CopyDocument /></el-icon> 复制所有链接
              </button>
              <button class="btn-brutal btn-open" @click="batchResultDialogVisible = false">
                 关闭
              </button>
           </div>
        </div>
      </div>
    </div>


    <!-- 自定义预览遮罩 (Brutalist Overlay) -->
    <div v-if="previewVisible" class="preview-overlay" @click.self="closePreview">
      
      <!-- 视频预览 -->
      <div v-if="previewType === 'video'" class="window-frame active" id="view-video" style="width: 800px; max-width: 90vw;">
        <div class="window-header">
            <span>媒体播放器 // {{ previewExtension.toUpperCase() }}</span>
            <div class="window-controls"><button @click="closePreview">X</button></div>
        </div>
        <div class="video-wrapper">
            <!-- 录制状态装饰 -->
            <div class="rec-overlay">
                <div class="rec-dot"></div>
                <div class="rec-text">REC</div>
            </div>
            <video 
              :src="previewUrl" 
              controls 
              autoplay
              referrerpolicy="no-referrer"
              class="brutal-video"
            >
              您的浏览器不支持视频标签。
            </video>
        </div>
        <div class="video-controls-bar">
            <div class="time-code">{{ videoTimeCode }}</div>
            <div>
                <button class="btn-brutal" @click="downloadFile(previewFileItem!)" style="padding: 5px 15px; font-size: 12px;">下载原片</button>
            </div>
        </div>
      </div>

      <!-- 图片预览 -->
      <div v-else-if="previewType === 'image'" class="window-frame active" id="view-image" style="width: auto; max-width: 90vw; min-width: 500px;">
        <div class="window-header">
            <span>图像查看器 // {{ previewExtension.toUpperCase() }}</span>
            <div class="window-controls"><button @click="closePreview">X</button></div>
        </div>
        <div class="image-stage">
            <img 
              :src="previewUrl" 
              class="preview-img" 
              referrerpolicy="no-referrer"
              @load="onImageLoad"
              @error="previewError = true"
            >
        </div>
        <div class="file-info-bar">
            <div class="file-info-item"><span>格式</span>{{ previewExtension.toUpperCase() }}</div>
            <div class="file-info-item"><span>分辨率</span>{{ imgResolution }}</div>
            <div class="file-info-item"><span>大小</span>{{ formatSize(previewFileItem?.size || 0) }}</div>
        </div>
        <div style="padding: 10px; background: white; display:flex; gap:10px;">
             <button class="btn-brutal" style="width:100%; background: white; color: black; border: 2px solid black;" @click="openExternal(previewUrl)">全屏查看</button>
             <button class="btn-brutal" style="width:100%; background: var(--acid-green); color: black;" @click="downloadFile(previewFileItem!)">下载图片</button>
        </div>
      </div>

      <!-- 其他类型兜底 (Text/PDF/Audio) 依然使用简单窗口包裹 -->
      <div v-else class="window-frame active" style="width: 800px; height: 80vh;">
          <div class="window-header">
            <span>预览 // {{ previewExtension.toUpperCase() }}</span>
            <div class="window-controls"><button @click="closePreview">X</button></div>
          </div>
          <div class="content-wrapper" style="flex: 1; overflow: auto; padding: 0; background: #eee;">
             <iframe v-if="previewType === 'pdf'" :src="previewUrl" style="width:100%; height:100%; border:none;"></iframe>
             <audio v-else-if="previewType === 'audio'" :src="previewUrl" controls style="margin: 50px auto; display:block;"></audio>
             <div v-else-if="previewType === 'text'" style="padding: 20px;">
                <pre v-if="textContent" style="white-space: pre-wrap; word-wrap: break-word;">{{ textContent }}</pre>
                <div v-else>载入中...</div>
             </div>
             <div v-else style="padding: 50px; text-align: center;">此文件类型不支持预览</div>
          </div>
      </div>

    </div>

    <!-- 分享结果弹窗 (Share Result Overlay) -->
    <div v-if="shareResultVisible" class="preview-overlay" @click.self="closeShareResult">
      <div class="window-frame active" style="width: 500px; max-width: 95vw;">
        <div class="window-header">
           <span>链接生成器_V2</span>
           <div class="window-controls"><button @click="closeShareResult">X</button></div>
        </div>
        
        <div class="share-receipt">
           <!-- 成功横幅 -->
           <div class="success-banner-box">
              <div class="star-icon">★</div>
              <span>链接已生成</span>
              <div class="star-icon">★</div>
           </div>

           <!-- 链接显示区域 -->
           <div class="link-section">
              <div class="brutal-label">公开链接地址</div>
              <div class="dashed-box url-box">
                 {{ shareResult.url }}
              </div>
           </div>

           <!-- 信息网格 -->
           <div class="info-grid">
              <div class="info-cell">
                 <div class="info-label">有效期</div>
                 <div class="info-value">{{ shareResult.expire ? formatDateSimple(shareResult.expire) : '永久有效' }}</div>
              </div>
              <div class="info-cell">
                 <div class="info-label">访问提取码</div>
                 <div class="info-value password-stamp">
                    {{ shareResult.password || '公开' }}
                 </div>
              </div>
           </div>

           <!-- 操作按钮组 (经过优化的配色) -->
           <div class="action-buttons-row">
              <button class="btn-brutal btn-copy" @click="copyLink(shareResult.code)">
                 <el-icon><CopyDocument /></el-icon> 复制链接
              </button>
              <button class="btn-brutal btn-open" @click="openExternal(shareResult.url)">
                 直接打开 <el-icon><TopRight /></el-icon>
              </button>
           </div>
           
           <div class="receipt-footer">
              // 生成时间: {{ new Date().toLocaleTimeString() }}
           </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useUIStore } from '../../stores/ui'

import {
  Refresh, Grid, List,
  Picture, VideoPlay, Headset, Document as DocIcon,
  CopyDocument, TopRight
} from '@element-plus/icons-vue'
import { fileApi, shareApi } from '../../api'

interface FileInfo {
  id: string
  fid: string
  name: string
  size: number
  is_dir: boolean
  mod_time: string
  parent_id: string
}

const loading = ref(false)
const files = ref<FileInfo[]>([])
const currentDirId = ref('')
const breadcrumbs = ref<{ id: string; name: string }[]>([])
const selectedFiles = ref<FileInfo[]>([])

const uiStore = useUIStore()

// 视图和排序
const viewMode = ref<'list' | 'grid'>('list')
const sortBy = ref<'name' | 'size' | 'time'>('name')
const sortOrder = ref<'asc' | 'desc'>('asc')

// 键盘事件监听
const handleKeydown = (e: KeyboardEvent) => {
  // 如果正在使用输入框，不触发快捷键
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
    return
  }

  // Esc 关闭预览
  if (e.key === 'Escape' && previewVisible.value) {
    closePreview()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  loadFiles()
  
  // 启动时码更新计时器
  startVideoTimer()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  stopVideoTimer()
})

const sortLabels: Record<string, string> = {
  name: '名称',
  size: '大小',
  time: '时间'
}

// 过滤和排序后的文件列表
const filteredFiles = computed(() => {
  let result = [...files.value]

  // 排序（文件夹始终在前）
  result.sort((a, b) => {
    if (a.is_dir && !b.is_dir) return -1
    if (!a.is_dir && b.is_dir) return 1

    let cmp = 0
    switch (sortBy.value) {
      case 'name':
        cmp = a.name.localeCompare(b.name, 'zh-CN')
        break
      case 'size':
        cmp = (a.size || 0) - (b.size || 0)
        break
      case 'time':
        cmp = new Date(a.mod_time || 0).getTime() - new Date(b.mod_time || 0).getTime()
        break
    }
    return sortOrder.value === 'asc' ? cmp : -cmp
  })

  return result
})

function handleSort(command: string) {
  if (sortBy.value === command) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = command as any
    sortOrder.value = 'asc'
  }
}

const shareDialogVisible = ref(false)
const creating = ref(false)
const shareForm = ref({
  target_type: 'file' as 'file' | 'folder',
  target_id: '',
  target_name: '',
  target_path: '',
  share_code: '',
  password: '',
  expire_at: null as Date | null,
  description: ''
})

// 批量分享
const batchShareDialogVisible = ref(false)
const batchCreating = ref(false)
const batchShareForm = ref({
  password: '',
  expire_at: null as Date | null
})
const batchResultDialogVisible = ref(false)
const batchResults = ref<any[]>([])

// 单个文件分享结果弹窗状态
const shareResultVisible = ref(false)
const shareResult = ref({
  url: '',
  code: '',
  password: '',
  expire: ''
})

function closeShareResult() {
  shareResultVisible.value = false
}

function formatDateSimple(dateStr: string) {
  if (!dateStr) return '永久'
  const date = new Date(dateStr)
  const now = new Date()
  const diffTime = Math.abs(date.getTime() - now.getTime())
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24)) 
  
  if (diffDays > 3650) return '永久' // 简单判定
  return `${diffDays} 天后过期`
}

// 预览相关
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewType = ref<'image' | 'video' | 'audio' | 'pdf' | 'text' | 'none'>('none')
const previewUrl = ref('')
const previewFileName = ref('')
const previewExtension = ref('')
const previewFileItem = ref<FileInfo | null>(null)
const previewError = ref(false)
const textContent = ref('')
// 图片/视频 额外数据
const imgResolution = ref('Loading...')
const videoTimeCode = ref('00:00:00:00')
let videoTimer: any = null

function startVideoTimer() {
    stopVideoTimer()
    videoTimer = setInterval(() => {
        const now = new Date()
        const ms = String(Math.floor(now.getMilliseconds() / 10)).padStart(2, '0')
        videoTimeCode.value = now.toTimeString().split(' ')[0] + ':' + ms
    }, 40) // 25fps simulation
}

function stopVideoTimer() {
    if (videoTimer) clearInterval(videoTimer)
}

function onImageLoad(e: Event) {
    const img = e.target as HTMLImageElement
    imgResolution.value = `${img.naturalWidth}x${img.naturalHeight}`
}

function closePreview() {
    previewVisible.value = false
    previewUrl.value = ''
    textContent.value = ''
    previewType.value = 'none'
}


async function loadFiles() {
  loading.value = true
  try {
    const res = await fileApi.list(currentDirId.value)
    files.value = res.data.files || []
    selectedFiles.value = []
  } catch (error) {
    // 错误已处理
  } finally {
    loading.value = false
  }
}

function handleRowClick(row: FileInfo) {
  if (row.is_dir) {
    navigateTo(row.id, row.name)
  }
}

function handleSelectionChange(selection: FileInfo[]) {
  selectedFiles.value = selection
}

function toggleSelect(file: FileInfo) {
  const index = selectedFiles.value.indexOf(file)
  if (index >= 0) {
    selectedFiles.value.splice(index, 1)
  } else {
    selectedFiles.value.push(file)
  }
}

function navigateTo(dirId: string, name?: string) {
  if (dirId === '') {
    breadcrumbs.value = []
  } else if (name) {
    breadcrumbs.value.push({ id: dirId, name })
  } else {
    const index = breadcrumbs.value.findIndex(b => b.id === dirId)
    if (index >= 0) {
      breadcrumbs.value = breadcrumbs.value.slice(0, index + 1)
    }
  }
  currentDirId.value = dirId
  loadFiles()
}

function openShareDialog(file: FileInfo) {
  shareForm.value = {
    target_type: file.is_dir ? 'folder' : 'file',
    target_id: file.is_dir ? file.id : (file.fid || file.id),
    target_name: file.name,
    target_path: breadcrumbs.value.map(b => b.name).join('/') + '/' + file.name,
    share_code: '',
    password: '',
    expire_at: null,
    description: ''
  }
  shareDialogVisible.value = true
}

function canPreview(fileName: string): boolean {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const previewableExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'mp4', 'webm', 'mp3', 'wav', 'pdf', 'txt', 'md', 'json', 'xml', 'yaml', 'yml', 'log']
  return previewableExts.includes(ext)
}

function getPreviewType(fileName: string): typeof previewType.value {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'ico', 'bmp']
  const videoExts = ['mp4', 'webm', 'ogg', 'm4v']
  const audioExts = ['mp3', 'wav', 'ogg', 'flac', 'm4a', 'aac']
  const textExts = ['txt', 'md', 'json', 'xml', 'yaml', 'yml', 'log', 'css', 'js', 'html', 'go', 'py', 'java']
  const pdfExts = ['pdf']

  if (imageExts.includes(ext)) return 'image'
  if (videoExts.includes(ext)) return 'video'
  if (audioExts.includes(ext)) return 'audio'
  if (textExts.includes(ext)) return 'text'
  if (pdfExts.includes(ext)) return 'pdf'
  return 'none'
}

async function openPreview(file: FileInfo) {
  if (file.is_dir) return
  previewFileItem.value = file
  previewFileName.value = file.name
  previewExtension.value = file.name.split('.').pop() || 'unknown'
  previewType.value = getPreviewType(file.name)
  previewUrl.value = ''
  previewError.value = false
  textContent.value = ''
  imgResolution.value = 'Analyzing...' // reset
  
  previewVisible.value = true
  previewLoading.value = true

  try {
    const fid = file.fid || file.id
    const res = await fileApi.link(fid)
    previewUrl.value = res.data.url

    if (previewType.value === 'text') {
      fetch(previewUrl.value)
        .then(res => res.text())
        .then(text => { textContent.value = text })
        .catch(() => { previewError.value = true })
    }
  } catch (error) {
    previewError.value = true
    uiStore.showToast('error', '预览错误', '无法获取文件链接')
  } finally {
    previewLoading.value = false
  }
}

async function downloadFile(file: FileInfo) {
  if (file.is_dir) return
  uiStore.showToast('success', '下载已开始', '正在请求安全节点...')
  try {
    const fid = file.fid || file.id
    const res = await fileApi.link(fid)
    openExternal(res.data.url)
  } catch (error) {
    uiStore.showToast('error', '下载失败', '获取链接失败 (503)')
  }
}

function openExternal(url: string) {
  const link = document.createElement('a')
  link.href = url
  link.target = '_self'
  link.rel = 'noreferrer'
  link.referrerPolicy = 'no-referrer'
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

async function createShare() {
  creating.value = true
  try {
    const data: any = {
      target_type: shareForm.value.target_type,
      target_id: shareForm.value.target_id,
      target_name: shareForm.value.target_name,
      target_path: shareForm.value.target_path
    }
    if (shareForm.value.share_code) {
      data.share_code = shareForm.value.share_code
    }
    if (shareForm.value.password) {
      data.password = shareForm.value.password
    }
    if (shareForm.value.expire_at) {
      data.expire_at = shareForm.value.expire_at.toISOString()
    }
    if (shareForm.value.description) {
      data.description = shareForm.value.description
    }

    const res = await shareApi.create(data)
    
    // 成功后显示自定义 Brutalist 弹窗，不再使用 Toast
    shareResult.value = {
      url: `${window.location.origin}/s/${res.data.share_code}`,
      code: res.data.share_code,
      password: data.password || '',
      expire: data.expire_at || ''
    }
    
    shareDialogVisible.value = false
    shareResultVisible.value = true
    
  } catch (error) {
    // 错误已处理
  } finally {
    creating.value = false
  }
}

function openBatchShareDialog() {
  batchShareForm.value = {
    password: '',
    expire_at: null
  }
  batchShareDialogVisible.value = true
}

async function createBatchShare() {
  batchCreating.value = true
  try {
    const items = selectedFiles.value.map(file => ({
      target_type: file.is_dir ? 'folder' : 'file' as 'file' | 'folder',
      target_id: file.is_dir ? file.id : (file.fid || file.id),
      target_name: file.name,
      target_path: breadcrumbs.value.map(b => b.name).join('/') + '/' + file.name,
      password: batchShareForm.value.password || undefined,
      expire_at: batchShareForm.value.expire_at?.toISOString()
    }))

    const res = await shareApi.batchCreate(items)
    batchResults.value = res.data || []
    batchShareDialogVisible.value = false
    batchResultDialogVisible.value = true
    selectedFiles.value = []
    uiStore.showToast('success', '批量分享成功', (res as any).message || '链接已生成')
  } catch (error) {
    // 错误已处理
  } finally {
    batchCreating.value = false
  }
}

// 文件类型图标和颜色
function getExtIcon(fileName: string) {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp']
  const videoExts = ['mp4', 'webm', 'avi', 'mov', 'mkv']
  const audioExts = ['mp3', 'wav', 'flac', 'm4a', 'aac']
  
  if (imageExts.includes(ext)) return Picture
  if (videoExts.includes(ext)) return VideoPlay
  if (audioExts.includes(ext)) return Headset
  return DocIcon
}

function getShareUrl(code: string): string {
  return `${window.location.origin}/s/${code}`
}

function copyLink(code: string) {
  const url = getShareUrl(code)
  navigator.clipboard.writeText(url)
  uiStore.showToast('success', '复制成功', '链接已复制到剪贴板')
}

function copyAllLinks() {
  const links = batchResults.value.map(r => getShareUrl(r.share_code)).join('\n')
  navigator.clipboard.writeText(links)
  uiStore.showToast('success', '全部复制成功', '链接已复制到剪贴板')
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(dateStr: string): string {
  if (!dateStr || dateStr.startsWith('0001') || dateStr.startsWith('1/1/1')) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  loadFiles()
})
</script>

<style scoped>
.files-page {
  max-width: 1400px;
}

.breadcrumb-card {
  margin-bottom: 20px;
}

.breadcrumb-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.breadcrumb-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.files-card {
  min-height: 400px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
  flex-wrap: wrap;
  gap: 12px;
}

.card-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.file-name {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.file-name:hover {
  color: #409eff;
}

.empty-state {
  padding: 60px 0;
}

/* 网格视图 */
.grid-view {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 16px;
  padding: 16px 0;
}

/* ====================
   预览 Overlay 核心样式 (Ported from 细节 1.html)
   ==================== */
.preview-overlay {
    position: fixed; top: 0; left: 0; width: 100%; height: 100%;
    background: rgba(0,0,0,0.8); backdrop-filter: blur(5px);
    z-index: 9999;
    display: flex; justify-content: center; align-items: center;
    animation: fade-in 0.2s;
}
@keyframes fade-in { from { opacity: 0; } to { opacity: 1; } }

.window-frame {
    background: var(--pure-white, white);
    border: 4px solid var(--pure-black, black);
    box-shadow: 6px 6px 0px var(--pure-black, black);
    display: flex; flex-direction: column;
    animation: slam-in 0.2s cubic-bezier(0.18, 0.89, 0.32, 1.28) forwards;
    /* max-height: 90vh; */
}
@keyframes slam-in { 0% { transform: scale(0.9) skewX(5deg); opacity: 0; } 100% { transform: scale(1) skewX(0); opacity: 1; } }

.window-header {
    background: var(--pure-black, black); color: var(--acid-green, #ccff00);
    padding: 10px 15px; font-weight: 900; text-transform: uppercase; font-size: 18px;
    border-bottom: 4px solid black; display: flex; justify-content: space-between; align-items: center;
}
.window-controls button {
    background: var(--acid-pink, #ff00ff); border: 2px solid white; color: white;
    width: 25px; height: 25px; font-weight: bold; margin-left: 5px; box-shadow: 2px 2px 0 black;
    cursor: pointer;
}

.btn-brutal {
    background: var(--pure-black); color: var(--pure-white);
    border: none; padding: 15px 20px; font-size: 16px; font-weight: 900;
    text-transform: uppercase; letter-spacing: 1px; transition: transform 0.1s;
    cursor: pointer;
}
.btn-brutal:hover {
    background: var(--acid-pink, #ff00ff) !important; color: var(--pure-black) !important;
    box-shadow: 4px 4px 0 var(--pure-black); transform: translate(-2px, -2px);
}
.btn-brutal:active { transform: translate(0, 0); box-shadow: none; }

/* 视频具体样式 */
.video-wrapper {
    width: 100%;
    background: black;
    position: relative;
    border-bottom: 4px solid black;
    line-height: 0;
}
.brutal-video {
    width: 100%;
    max-height: 70vh;
    outline: none;
}
.rec-overlay {
    position: absolute; top: 15px; left: 15px;
    display: flex; align-items: center; gap: 8px;
    pointer-events: none; z-index: 5;
}
.rec-dot {
    width: 12px; height: 12px; background: red; border-radius: 50%;
    animation: blink 1s infinite;
}
.rec-text { color: red; font-weight: bold; font-family: monospace; letter-spacing: 2px; }
@keyframes blink { 50% { opacity: 0; } }

.video-controls-bar {
    padding: 10px; background: white;
    display: flex; justify-content: space-between; align-items: center;
}
.time-code { font-family: monospace; font-size: 14px; font-weight: bold; }

/* 图片具体样式 */
.image-stage {
    background-color: #eee;
    background-image: linear-gradient(45deg, #ccc 25%, transparent 25%), linear-gradient(-45deg, #ccc 25%, transparent 25%), linear-gradient(45deg, transparent 75%, #ccc 75%), linear-gradient(-45deg, transparent 75%, #ccc 75%);
    background-size: 20px 20px;
    background-position: 0 0, 0 10px, 10px -10px, -10px 0px;
    
    width: 100%; min-height: 300px;
    display: flex; align-items: center; justify-content: center;
    border-bottom: 4px solid black;
    padding: 20px;
    overflow: hidden;
}
.preview-img {
    max-width: 100%; max-height: 70vh;
    border: 2px solid black;
    box-shadow: 4px 4px 0 rgba(0,0,0,0.2);
}
.file-info-bar {
    background: black; color: white;
    padding: 12px;
    display: grid; grid-template-columns: 1fr 1fr 1fr;
    font-family: monospace; font-size: 12px;
    text-align: center;
}
.file-info-item span { color: var(--acid-green, #ccff00); display: block; font-weight: bold; }


.grid-item {
  background: var(--card-bg, white);
  border: 1px solid var(--border-color, #dcdfe6);
  border-radius: 12px;
  padding: 16px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
  position: relative;
}

.grid-item:hover {
  border-color: #409eff;
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
}

.grid-item.selected {
  border-color: #409eff;
  background: #ecf5ff;
}

.grid-checkbox {
  position: absolute;
  top: 8px;
  left: 8px;
  opacity: 0;
  transition: opacity 0.3s;
}

.grid-item:hover .grid-checkbox,
.grid-item.selected .grid-checkbox {
  opacity: 1;
}

.grid-icon {
  margin-bottom: 12px;
}

/* Neo-Brutalist Form Styles */
.brutal-form-content {
  padding: 25px;
  background: white;
}

.form-group {
  margin-bottom: 20px;
}

.brutal-label-tag {
  background: black;
  color: var(--acid-green);
  padding: 4px 8px;
  font-family: 'Helvetica Neue', sans-serif;
  font-weight: 900;
  font-size: 12px;
  margin-bottom: 8px;
  display: inline-block;
  text-transform: uppercase;
  box-shadow: 4px 4px 0 rgba(0,0,0,0.1);
  transform: skewX(-5deg);
}

.static-value-box {
  background: #f0f0f0;
  border: var(--border-thick);
  padding: 12px;
  font-family: monospace;
  color: #666;
  font-weight: bold;
}

.input-group-brutal {
  display: flex;
  align-items: stretch;
}

.input-group-brutal .prefix {
  background: black;
  color: white;
  display: flex;
  align-items: center;
  padding: 0 15px;
  font-family: monospace;
  font-weight: bold;
  border: var(--border-thick);
  border-right: none;
}

.brutal-input-reset {
  width: 100%;
  padding: 12px 15px;
  border: var(--border-thick);
  background: white;
  font-family: monospace;
  font-size: 14px;
  outline: none;
  transition: all 0.2s;
}

.brutal-input-reset:focus {
  background: var(--acid-green);
  box-shadow: 4px 4px 0 black;
  transform: translate(-2px, -2px);
}

.date-picker-wrapper-brutal :deep(.el-input__wrapper) {
  border-radius: 0;
  box-shadow: none !important;
  border: 4px solid black;
  padding: 5px 10px;
}
.date-picker-wrapper-brutal :deep(.el-input__wrapper:hover) {
  box-shadow: none !important;
}
.date-picker-wrapper-brutal :deep(.el-input__wrapper.is-focus) {
  background: var(--acid-green) !important;
}

.form-actions-brutal {
  display: flex;
  gap: 15px;
  margin-top: 30px;
  padding-top: 20px;
  border-top: 2px dashed #ccc;
}

.btn-cancel {
  flex: 1;
  background: white;
  color: black;
  border: 4px solid black;
}
.btn-cancel:hover {
  background: #eee;
  box-shadow: 4px 4px 0 black;
}

.btn-confirm {
  flex: 2;
  background: black;
  color: var(--acid-green);
  border: 4px solid black;
}
.btn-confirm:hover {
  background: var(--acid-cyan);
  color: black;
  box-shadow: 6px 6px 0 rgba(0,0,0,0.2);
}

/* Share Receipt Modal Styles */
.share-receipt {
  padding: 25px;
  background: #f0f0f0; /* Light texture bg */
  background-image: radial-gradient(#999 1px, transparent 1px);
  background-size: 20px 20px;
}

.success-banner-box {
  background: var(--acid-green);
  border: var(--border-thick);
  padding: 15px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 24px;
  font-weight: 900;
  margin-bottom: 25px;
  box-shadow: 4px 4px 0 black;
  transform: rotate(-1deg);
}

.link-section {
  margin-bottom: 25px;
}

.dashed-box {
  background: white;
  border: 2px dashed black;
  padding: 15px;
  font-family: 'Courier New', monospace;
  font-weight: bold;
  word-break: break-all;
  font-size: 16px;
  margin-top: 5px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 30px;
}

.info-cell {
  background: white;
  border: var(--border-thick);
  padding: 15px;
}

.info-label {
  font-size: 12px;
  color: #666;
  margin-bottom: 5px;
  font-weight: bold;
  text-transform: uppercase;
}

.info-value {
  font-size: 20px;
  font-weight: 900;
}

.password-stamp {
  background: var(--acid-pink);
  color: white;
  display: inline-block;
  padding: 2px 8px;
  transform: rotate(2deg);
  border: 2px solid black;
}

.action-buttons-row {
  display: flex;
  gap: 15px;
}

.btn-copy {
  flex: 3;
  background: var(--pure-black);
  color: var(--acid-green);
  border: 4px solid black; /* Keep border for structure */
}
.btn-copy:hover {
  background: var(--acid-green);
  color: black;
  box-shadow: 4px 4px 0 rgba(0,0,0,0.5);
}

.btn-open {
  flex: 2;
  background: white;
  color: black;
  border: 4px solid black;
}
.btn-open:hover {
  background: var(--acid-pink);
  color: white;
  box-shadow: 4px 4px 0 black;
}

.receipt-footer {
  margin-top: 20px;
  font-family: monospace;
  text-align: right;
  font-size: 10px;
  opacity: 0.5;
}

.grid-name {
  font-size: 13px;
  color: var(--text-primary, #303133);
  word-break: break-all;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.4;
  min-height: 36px;
}

.grid-size {
  font-size: 11px;
  color: var(--text-secondary, #909399);
  margin-top: 4px;
}

.grid-actions {
  position: absolute;
  bottom: -10px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 8px;
  opacity: 0;
  transition: all 0.3s;
}

.grid-item:hover .grid-actions {
  opacity: 1;
  bottom: 8px;
}

.rotate-180 {
  transform: rotate(180deg);
}

/* 批量分享 */
.batch-share-info {
  padding: 8px 0;
}

.batch-file-list {
  max-height: 200px;
  overflow-y: auto;
  margin: 16px 0;
  padding: 12px;
  background: var(--main-bg, #f5f7fa);
  border-radius: 8px;
}

.batch-file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 14px;
}

.batch-form {
  margin-top: 20px;
}

/* 预览样式 */
.preview-content {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

.preview-image {
  max-width: 100%;
  max-height: 70vh;
  border-radius: 8px;
}

.preview-video {
  width: 100%;
  max-height: 80vh;
}

.preview-audio {
  width: 100%;
}

.preview-pdf {
  width: 100%;
  height: 70vh;
  border: none;
}

.preview-text {
  width: 100%;
  max-height: 70vh;
  overflow: auto;
  background: #1e1e1e;
  border-radius: 8px;
  padding: 16px;
}

.preview-text pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'Fira Code', monospace;
  font-size: 14px;
  white-space: pre-wrap;
  word-break: break-all;
}

.preview-error {
  text-align: center;
  color: #909399;
}

.preview-error p {
  margin-top: 16px;
}

/* 批量结果 */
.batch-result {
  padding: 8px 0;
}

.result-alert {
  margin-bottom: 16px;
}

.result-actions {
  margin-top: 20px;
  text-align: center;
}

/* Batch Styling */
.batch-info-box {
  background: var(--acid-cyan);
  border: var(--border-thin);
  padding: 10px;
  font-family: monospace;
  margin-bottom: 15px;
}

.batch-file-list-brutal {
  border: 2px dashed #ccc;
  padding: 10px;
  max-height: 150px;
  overflow-y: auto;
  background: #f9f9f9;
}

.batch-file-item-brutal {
  padding: 5px;
  border-bottom: 1px solid #eee;
  font-family: monospace;
  font-size: 12px;
}

/* Batch Results */
.batch-result-list {
  border: var(--border-thick);
  background: white;
}

.batch-result-header {
  background: black;
  color: white;
  display: flex;
  padding: 8px;
  font-family: monospace;
  font-weight: bold;
  font-size: 12px;
}

.batch-result-scroll {
  max-height: 300px;
  overflow-y: auto;
}

.batch-result-row {
  display: flex;
  align-items: center;
  padding: 8px;
  border-bottom: 1px solid #eee;
  font-family: monospace;
  font-size: 12px;
}
.batch-result-row:hover {
  background: #f0f0f0;
}

.res-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 10px;
}

.res-link {
  flex: 2;
  color: var(--acid-pink);
  font-weight: bold;
}

.btn-icon-tiny {
  background: black;
  color: white;
  border: none;
  width: 24px;
  height: 24px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.btn-icon-tiny:hover {
  background: var(--acid-green);
  color: black;
}
</style>
