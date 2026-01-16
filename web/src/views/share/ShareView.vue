<template>
  <div class="share-page-wrapper">
    <!-- 密码验证页面 (Encryption Interface) -->
    <div v-if="needPassword && !verified" class="window-frame active centered-window">
      <div class="window-header">
        <span>⚠️ 访问受限</span>
        <div class="window-controls"><button>X</button></div>
      </div>
      <div class="content-area centered-content">
         <div class="glitch-text">拒绝访问</div>
         <p class="brutal-label" style="text-align: center; margin-bottom: 20px;">检测到加密归档</p>
         
         <input
           v-model="password"
           type="password"
           placeholder="请输入提取码..."
           class="brutal-input sound-type"
           style="border-color: red; color: red;"
           @keyup.enter="verifyPassword"
         />
         
         <button
           class="btn-brutal sound-click"
           style="width: 100%; background: red; color: white;"
           @click="verifyPassword"
         >
           {{ verifying ? '解密中...' : '解密档案' }}
         </button>
      </div>
    </div>

    <!-- 分享内容页面 -->
    <div 
      v-else-if="shareInfo" 
      class="window-frame active"
      :class="shareInfo.target_type === 'folder' ? 'huge-window' : 'compact-window'"
    >
      <div class="window-header">
        <span>数据传输节点</span>
        <div class="window-controls"><button>_</button><button>X</button></div>
      </div>
      
      <div class="content-area">
        <!-- Header Info -->
        <div class="file-header-brutal" :class="{ 'compact-header': shareInfo.target_type !== 'folder' }">
          <div class="file-icon-box">
             <el-icon :size="40" :color="'#000'">
                <Folder v-if="shareInfo.target_type === 'folder'" />
                <component v-else :is="getFileIcon()" />
             </el-icon>
          </div>
          <div class="file-info-text">
            <h1 class="file-title">{{ shareInfo.target_name }}</h1>
            <div class="tag">类型: {{ shareInfo.target_type === 'folder' ? '目录' : '文件' }}</div>
          </div>
        </div>

        <!-- Folder Content -->
        <div v-if="shareInfo.target_type === 'folder'" class="folder-content">
           <!-- 下载次数限制提示 -->
           <div v-if="shareInfo.max_downloads && shareInfo.max_downloads > 0" class="download-limit-info folder-limit-info">
               <span v-if="!shareInfo.download_exhausted">
                   剩余下载次数：{{ shareInfo.remaining_downloads }} / {{ shareInfo.max_downloads }}
               </span>
               <span v-else class="exhausted-warning">下载次数已耗尽，无法继续下载</span>
           </div>
           <!-- Toolbar -->
           <div class="brutal-toolbar">
              <div class="nav-breadcrumbs">
                  <button class="nav-btn-simple" @click="navigateFolder('')">根目录</button>
                  <span v-for="(item, index) in folderBreadcrumbs" :key="index">
                      <span>/</span>
                      <button class="nav-btn-simple" @click="navigateFolder(item.id)">{{ item.name }}</button>
                  </span>
              </div>
              
              <div class="toolbar-actions">
                  <input 
                    v-model="searchKeyword" 
                    placeholder="检索..." 
                    class="brutal-input mini-input"
                  >
                  <button class="btn-brutal mini-btn" @click="viewMode = viewMode === 'list' ? 'grid' : 'list'">
                    {{ viewMode === 'list' ? '[网格]' : '[列表]' }}
                  </button>
              </div>
           </div>

           <!-- List View (Terminal Style) -->
           <div v-if="viewMode === 'list'" class="file-browser">
                <div class="file-header-row">
                    <span style="flex: 2">文件名</span>
                    <span style="flex: 1">大小</span>
                    <span style="flex: 1; text-align: right;">操作</span>
                </div>
                <div v-if="loadingFiles" class="file-item">载入数据...</div>
                <div 
                    v-else 
                    v-for="file in filteredFiles" 
                    :key="file.id" 
                    class="file-item sound-hover"
                    @click="handleFileClick(file)"
                >
                    <span style="flex: 2; display: flex; align-items: center; gap: 10px;">
                        <el-icon :size="16"><component :is="file.is_dir ? Folder : getExtIcon(file.name)" /></el-icon>
                        {{ file.name }}
                    </span>
                    <span style="flex: 1; font-family: monospace;">{{ file.is_dir ? '<DIR>' : formatSize(file.size) }}</span>
                    <span style="flex: 1; text-align: right;">
                        <button
                            v-if="!file.is_dir"
                            class="txt-btn"
                            :class="{ 'disabled': shareInfo?.download_exhausted }"
                            :disabled="shareInfo?.download_exhausted"
                            @click.stop="downloadFolderFile(file)"
                        >
                            {{ shareInfo?.download_exhausted ? '[已关闭]' : '[下载]' }}
                        </button>
                        <button v-if="!file.is_dir && canPreview(file.name)" class="txt-btn" @click.stop="previewFolderFile(file)">[预览]</button>
                    </span>
                </div>
                <div v-if="filteredFiles.length === 0 && !loadingFiles" class="file-item">空目录</div>
           </div>

           <!-- Grid View -->
           <div v-else v-loading="loadingFiles" class="grid-view-brutal">
              <div
                  v-for="file in filteredFiles"
                  :key="file.id"
                  class="grid-item-brutal"
                  @click="handleFileClick(file)"
                  @dblclick="file.is_dir ? null : downloadFolderFile(file)"
              >
                  <div class="grid-icon">
                     <el-icon :size="32"><component :is="file.is_dir ? Folder : getExtIcon(file.name)" /></el-icon>
                  </div>
                  <div class="grid-name">{{ file.name }}</div>
                  <div class="grid-size">{{ file.is_dir ? '<DIR>' : formatSize(file.size) }}</div>
              </div>
           </div>
        </div>

        <!-- Single File Actions -->
        <div v-else class="single-file-actions">
            <!-- 下载次数限制提示 -->
            <div v-if="shareInfo.max_downloads && shareInfo.max_downloads > 0" class="download-limit-info">
                <span v-if="!shareInfo.download_exhausted">
                    剩余下载次数：{{ shareInfo.remaining_downloads }} / {{ shareInfo.max_downloads }}
                </span>
                <span v-else class="exhausted-warning">下载次数已耗尽</span>
            </div>
            <!-- Compact Actions without giant icon -->
            <div class="action-buttons file-action-buttons">
                <button
                    class="btn-brutal sound-click primary-action"
                    :class="{ 'disabled': shareInfo.download_exhausted }"
                    :disabled="shareInfo.download_exhausted"
                    @click="downloadFile"
                >
                    {{ shareInfo.download_exhausted ? '下载已关闭' : '立即下载' }}
                </button>
                <button v-if="previewType !== 'none'" class="btn-brutal sound-click secondary-action" @click="showPreview = true">在线预览</button>
            </div>
        </div>

      </div>
      
      <div class="window-footer">
        {{ siteConfig.share_footer }}
      </div>
    </div>

    <!-- Error Page -->
    <div v-else-if="error" class="window-frame active centered-window error-window">
        <div class="window-header" style="background: red;">
            <span>系统错误</span>
        </div>
        <div class="content-area centered-content">
            <h2 style="color: red;">错误: {{ error }}</h2>
            <button class="btn-brutal" @click="$router.push('/')">返回首页</button>
        </div>
    </div>

    <!-- Loading -->
    <div v-else class="loading-overlay">
        <div class="loading-text">正在连接节点...</div>
    </div>

    <!-- Custom Preview Overlay for Share View -->
    <div v-if="showPreview" class="preview-overlay" @click.self="closePreview">
       <!-- 视频预览 -->
      <div v-if="previewType === 'video'" class="window-frame active" style="width: 800px; max-width: 90vw;">
        <div class="window-header">
            <span>媒体播放器 // {{ previewFileName.split('.').pop()?.toUpperCase() }}</span>
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
            <!-- 简单的实时时钟 -->
            <div class="time-code">{{ videoTimeCode }}</div>
            <div>
                <button class="btn-brutal" @click="downloadFile" style="padding: 5px 15px; font-size: 12px;">下载原片</button>
            </div>
        </div>
      </div>

      <!-- 图片预览 -->
      <div v-else-if="previewType === 'image'" class="window-frame active" style="width: auto; max-width: 90vw; min-width: 500px;">
        <div class="window-header">
            <span>图像查看器 // {{ previewFileName.split('.').pop()?.toUpperCase() }}</span>
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
            <div class="file-info-item"><span>格式</span>{{ previewFileName.split('.').pop()?.toUpperCase() }}</div>
            <div class="file-info-item"><span>分辨率</span>{{ imgResolution }}</div>
            <div class="file-info-item"><span>状态</span>ONLINE</div>
        </div>
        <div style="padding: 10px; background: white; display:flex; gap:10px;">
             <button class="btn-brutal" style="width:100%; background: white; color: black; border: 2px solid black;" @click="openExternal(previewUrl)">全屏查看</button>
             <button class="btn-brutal" style="width:100%; background: var(--acid-green); color: black;" @click="downloadFile">下载图片</button>
        </div>
      </div>

      <!-- 其他兜底 -->
       <div v-else class="window-frame active" style="width: 800px; height: 80vh;">
          <div class="window-header">
            <span>预览 // FILE</span>
            <div class="window-controls"><button @click="closePreview">X</button></div>
          </div>
          <div class="content-wrapper" style="flex: 1; overflow: auto; padding: 0; background: #eee;">
             <iframe v-if="previewType === 'pdf'" :src="previewUrl" style="width:100%; height:100%; border:none;"></iframe>
             <audio v-else-if="previewType === 'audio'" :src="previewUrl" controls style="margin: 50px auto; display:block;"></audio>
             <div v-else-if="previewType === 'text'" style="padding: 20px;">
                <pre v-if="textContent" style="white-space: pre-wrap; word-wrap: break-word;">{{ textContent }}</pre>
                <div v-else>载入中...</div>
             </div>
             <div v-if="previewError" style="padding: 50px; text-align: center; color: red;">预览加载失败</div>
          </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onUnmounted } from 'vue'
import { useUIStore } from '../../stores/ui'

import { useRoute } from 'vue-router'
import {
  Document, Folder,
  Picture, VideoPlay, Headset, Document as DocIcon
} from '@element-plus/icons-vue'
import { publicApi } from '../../api'

const uiStore = useUIStore() 


const route = useRoute()
const code = route.params.code as string

const shareInfo = ref<{
  share_code: string
  target_type: string
  target_name: string
  target_id?: string
  max_downloads?: number
  download_count?: number
  remaining_downloads?: number
  download_exhausted?: boolean
} | null>(null)

const needPassword = ref(false)
const verified = ref(false)
const password = ref('')
const verifying = ref(false)
const error = ref('')

// 站点配置
const siteConfig = ref({
  share_footer: 'Powered by WOOPEN_OS // BRUTAL_EDITION'
})

// 加载站点配置
async function loadSiteConfig() {
  try {
    const res = await publicApi.getSiteConfig()
    if (res.code === 0 && res.data) {
      siteConfig.value = {
        share_footer: res.data.share_footer || 'Powered by WOOPEN_OS // BRUTAL_EDITION'
      }
    }
  } catch (error) {
    console.log('加载站点配置失败，使用默认值')
  }
}

// Preview
const showPreview = ref(false)
const previewType = ref('none')
const previewFileName = ref('')
const previewUrl = ref('')
const previewError = ref(false)
const textContent = ref('')
const imgResolution = ref('Analyzing...')
const videoTimeCode = ref('00:00:00:00')
let videoTimer: any = null

function startVideoTimer() {
    stopVideoTimer()
    videoTimer = setInterval(() => {
        const now = new Date()
        const ms = String(Math.floor(now.getMilliseconds() / 10)).padStart(2, '0')
        videoTimeCode.value = now.toTimeString().split(' ')[0] + ':' + ms
    }, 40)
}
function stopVideoTimer() {
    if (videoTimer) clearInterval(videoTimer)
}

function onImageLoad(e: Event) {
    const img = e.target as HTMLImageElement
    imgResolution.value = `${img.naturalWidth}x${img.naturalHeight}`
}

function closePreview() {
    showPreview.value = false
    previewUrl.value = ''
    textContent.value = ''
    stopVideoTimer()
}


// Folder
const loadingFiles = ref(false)
const folderFiles = ref<any[]>([])
const folderBreadcrumbs = ref<{ id: string; name: string }[]>([])
const currentFolderId = ref('')

// View & Sort
const viewMode = ref<'list' | 'grid'>('list')
const searchKeyword = ref('')

const filteredFiles = computed(() => {
  let files = [...folderFiles.value]
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    files = files.filter(f => f.name.toLowerCase().includes(keyword))
  }
  // Simplified sorting
  files.sort((a, b) => {
    if (a.is_dir && !b.is_dir) return -1
    if (!a.is_dir && b.is_dir) return 1
    return a.name.localeCompare(b.name, 'zh-CN')
  })
  return files
})

async function loadShare() {
  try {
    const res = await publicApi.access(code)
    if (res.code === 0) {
      shareInfo.value = res.data
      needPassword.value = res.data.need_password
      if (!needPassword.value) {
        verified.value = true
        detectPreviewType()
        if (res.data.target_type === 'folder') {
          loadFolderFiles()
        }
      }
    } else {
      error.value = res.message || '链接失效或不存在'
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || '连接失败'
  }
}

async function verifyPassword() {
  if (!password.value) {
    alert('请输入提取码')
    return
  }
  verifying.value = true
  try {
    const res = await publicApi.verify(code, password.value)
    if (res.code === 0) {
      shareInfo.value = {
        ...shareInfo.value!,
        target_id: res.data.target_id
      }
      verified.value = true
      detectPreviewType()
      if (shareInfo.value?.target_type === 'folder') {
        loadFolderFiles()
      }
    } else {
      alert(res.message || '密码错误')
    }
  } catch (err: any) {
    alert(err.response?.data?.message || '验证失败')
  } finally {
    verifying.value = false
  }
}

function detectPreviewType() {
  if (!shareInfo.value) return
  const name = shareInfo.value.target_name
  const ext = name.split('.').pop()?.toLowerCase() || ''
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'ico', 'bmp']
  const videoExts = ['mp4', 'webm', 'ogg', 'm4v']
  const audioExts = ['mp3', 'wav', 'ogg', 'flac', 'm4a', 'aac']
  const textExts = ['txt', 'md', 'json', 'xml', 'yaml', 'yml', 'log', 'css', 'js', 'html', 'go', 'py', 'java']
  const pdfExts = ['pdf']

  if (imageExts.includes(ext)) previewType.value = 'image'
  else if (videoExts.includes(ext)) previewType.value = 'video'
  else if (audioExts.includes(ext)) previewType.value = 'audio'
  else if (textExts.includes(ext)) previewType.value = 'text'
  else if (pdfExts.includes(ext)) previewType.value = 'pdf'
  else previewType.value = 'none'
}

function downloadFile() {
  uiStore.showToast('success', '下载已开始', '数据传输请求已发送...')
  const url = publicApi.getDownloadUrl(code, undefined, needPassword.value ? password.value : undefined)
  openExternal(url)
}

async function loadFolderFiles(dirId?: string) {
  loadingFiles.value = true
  try {
    const res = await publicApi.getFiles(code, dirId, needPassword.value ? password.value : undefined)
    if (res.code === 0) {
      folderFiles.value = res.data.files || []
      currentFolderId.value = res.data.current_id
      // 更新下载限制信息
      if (shareInfo.value) {
        shareInfo.value.max_downloads = res.data.max_downloads
        shareInfo.value.download_count = res.data.download_count
        shareInfo.value.remaining_downloads = res.data.remaining_downloads
        shareInfo.value.download_exhausted = res.data.download_exhausted
      }
    }
  } catch (err) { } finally {
    loadingFiles.value = false
  }
}

function navigateFolder(dirId: string) {
  searchKeyword.value = ''
  if (dirId === '') {
    folderBreadcrumbs.value = []
    loadFolderFiles()
  } else {
    const index = folderBreadcrumbs.value.findIndex(b => b.id === dirId)
    if (index >= 0) {
      folderBreadcrumbs.value = folderBreadcrumbs.value.slice(0, index + 1)
    }
    loadFolderFiles(dirId)
  }
}

function handleFileClick(file: any) {
  if (file.is_dir) {
    folderBreadcrumbs.value.push({ id: file.id, name: file.name })
    loadFolderFiles(file.id)
  }
}

function downloadFolderFile(file: any) {
  const fileId = file.fid || file.id
  uiStore.showToast('success', '下载已开始', '正在获取文件 [' + file.name + ']...')
  const url = publicApi.getDownloadUrl(code, fileId, needPassword.value ? password.value : undefined)
  openExternal(url)
}

function canPreview(fileName: string): boolean {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const previewableExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'mp4', 'webm', 'mp3', 'wav', 'pdf']
  return previewableExts.includes(ext)
}

function previewFolderFile(file: any) {
  previewFileName.value = file.name
  const fileId = file.fid || file.id
  previewUrl.value = publicApi.getPreviewUrl(code, fileId, needPassword.value ? password.value : undefined)
  
  const ext = file.name.split('.').pop()?.toLowerCase() || ''
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp']
  const videoExts = ['mp4', 'webm']
  const audioExts = ['mp3', 'wav']
  
  if (imageExts.includes(ext)) previewType.value = 'image'
  else if (videoExts.includes(ext)) previewType.value = 'video'
  else if (audioExts.includes(ext)) previewType.value = 'audio'
  else if (ext === 'pdf') previewType.value = 'pdf'
  
  showPreview.value = true
}

function openExternal(url: string) {
  const link = document.createElement('a')
  link.href = url
  link.target = '_self'
  link.rel = 'noreferrer'
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

function getExtIcon(fileName: string) {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp']
  const videoExts = ['mp4', 'webm', 'avi']
  const audioExts = ['mp3', 'wav']
  if (imageExts.includes(ext)) return Picture
  if (videoExts.includes(ext)) return VideoPlay
  if (audioExts.includes(ext)) return Headset
  return DocIcon
}

function getFileIcon() {
  return Document
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

watch(showPreview, (val) => {
  if (val && shareInfo.value?.target_type !== 'folder') {
    previewFileName.value = shareInfo.value?.target_name || ''
    previewUrl.value = publicApi.getPreviewUrl(code, undefined, needPassword.value ? password.value : undefined)
    previewError.value = false
    if (previewType.value === 'text') {
      fetch(previewUrl.value)
        .then(res => res.text())
        .then(text => { textContent.value = text })
        .catch(() => { previewError.value = true })
    }
  }
})

// Listen specific key for share view
function handleKeydown(e: KeyboardEvent) {
   if (e.key === 'Escape' && showPreview.value) {
       closePreview()
   }
}

onMounted(() => {
  loadSiteConfig()
  loadShare()
  startVideoTimer()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
    stopVideoTimer()
    window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.share-page-wrapper {
    width: 100%;
    min-height: 100vh;
    padding: 20px;
    display: flex;
    justify-content: center;
    align-items: center;
}

.centered-window {
    width: 400px;
    background: white;
}

.huge-window {
    width: 1000px;
    height: 80vh; 
    background: white;
    display: flex;
    flex-direction: column;
}

.compact-window {
    width: 500px;
    background: white;
    display: flex;
    flex-direction: column;
    /* Auto height so it hugs content */
}

.centered-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px;
}

.glitch-text {
    font-size: 24px;
    font-weight: bold;
    color: red;
    margin-bottom: 10px;
    animation: text-flash 0.2s infinite;
}
@keyframes text-flash {
    0% { opacity: 1; }
    50% { opacity: 0.5; }
    100% { opacity: 1; }
}

/* File Header */
.file-header-brutal {
    display: flex;
    align-items: center;
    border-bottom: var(--border-thin);
    padding-bottom: 20px;
    margin-bottom: 20px;
}

.compact-header {
    flex-direction: column; /* Stack vertically for compact view */
    align-items: flex-start;
    padding-bottom: 10px; /* Tighter padding */
    margin-bottom: 15px;
}
.compact-header .file-icon-box {
    margin-bottom: 10px;
}

.file-icon-box {
    width: 60px;
    height: 60px;
    border: var(--border-thin);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 20px;
    box-shadow: 4px 4px 0 black;
    background: var(--acid-cyan);
}

.file-title {
    font-size: 24px;
    margin: 0 0 5px 0;
    text-transform: uppercase;
    word-break: break-all; /* Ensure long titles wrap */
}

/* Toolbar */
.brutal-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    flex-wrap: wrap;
    gap: 10px;
}

.nav-btn-simple {
    background: transparent;
    border: none;
    font-family: monospace;
    font-weight: bold;
    cursor: pointer;
    padding: 5px;
}
.nav-btn-simple:hover {
    background: var(--acid-green);
}

.toolbar-actions {
    display: flex;
    gap: 10px;
}

.mini-input {
    padding: 8px;
    margin: 0;
    width: 150px;
    font-size: 12px;
}

.mini-btn {
    padding: 8px 15px;
    font-size: 12px;
}

/* File Browser (Terminal List) */
.file-browser {
    border: var(--border-thick);
    background: white;
}

.file-header-row {
    background: black;
    color: white;
    display: flex;
    padding: 10px;
    font-family: monospace;
    font-weight: bold;
}

.file-item {
    display: flex;
    padding: 10px;
    border-bottom: 1px solid #ccc;
    cursor: pointer;
    align-items: center;
    font-family: monospace;
}
.file-item:hover {
    background: var(--acid-cyan);
}

.txt-btn {
    background: transparent;
    border: none;
    cursor: pointer;
    font-weight: bold;
    margin-left: 5px;
}
.txt-btn:hover {
    color: var(--acid-pink);
    text-decoration: underline;
    background: transparent;
}

/* Grid View */
.grid-view-brutal {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 15px;
}

.grid-item-brutal {
    border: var(--border-thin);
    padding: 10px;
    text-align: center;
    cursor: pointer;
    transition: 0.1s;
    background: white;
}
.grid-item-brutal:hover {
    box-shadow: 4px 4px 0 black;
    transform: translate(-2px, -2px);
    background: var(--acid-green);
}

.grid-name {
    font-size: 12px;
    font-weight: bold;
    margin-top: 5px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.grid-size {
    font-size: 10px;
    color: #666;
}

/* 下载次数限制提示 */
.download-limit-info {
    text-align: center;
    margin-bottom: 15px;
    font-family: monospace;
    font-weight: bold;
    padding: 10px;
    background: #f0f0f0;
    border: 2px solid black;
}

.folder-limit-info {
    margin-bottom: 20px;
}

.exhausted-warning {
    color: red;
    animation: text-flash 0.5s infinite;
}

.btn-brutal.disabled,
.txt-btn.disabled {
    opacity: 0.5 !important;
    cursor: not-allowed !important;
    pointer-events: none;
}

.btn-brutal.disabled {
    background: #ccc !important;
    box-shadow: none !important;
}

/* Single File Actions */
.single-file-actions {
    display: flex;
    flex-direction: column;
    margin-top: 10px;
    margin-bottom: 10px;
}

.file-action-buttons {
    display: flex;
    gap: 10px;
    width: 100%;
}

.primary-action {
    background: var(--acid-green) !important;
    color: black !important;
    flex: 1;
}

.secondary-action {
    flex: 1;
}

.window-footer {
    padding: 10px;
    background: #eee;
    font-family: monospace;
    text-align: right;
    font-size: 10px;
    border-top: var(--border-thick);
    margin-top: auto; 
}

/* Loading State */
.loading-overlay {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
}

.loading-text {
    background: black;
    color: var(--acid-green);
    padding: 20px;
    font-family: monospace;
    font-weight: bold;
    box-shadow: var(--shadow-hard);
}

/* Mobile */
@media (max-width: 768px) {
    /* 页面容器优化 - 固定一屏居中 */
    .share-page-wrapper {
        padding: 10px;
        padding-top: calc(10px + env(safe-area-inset-top, 0px));
        padding-bottom: calc(10px + env(safe-area-inset-bottom, 0px));
        min-height: 100dvh;
        height: 100dvh;
        overflow: hidden;
        box-sizing: border-box;
    }

    .huge-window, .compact-window, .centered-window {
        width: 100%;
        max-height: calc(100dvh - 20px - env(safe-area-inset-top, 0px) - env(safe-area-inset-bottom, 0px));
        overflow-y: auto;
    }

    .huge-window {
        height: auto;
    }

    /* 工具栏优化 */
    .brutal-toolbar {
        flex-direction: column;
        align-items: stretch;
        gap: 10px;
    }

    .nav-breadcrumbs {
        overflow-x: auto;
        white-space: nowrap;
        -webkit-overflow-scrolling: touch;
    }

    .toolbar-actions {
        width: 100%;
        flex-wrap: wrap;
    }

    .mini-input {
        flex: 1;
        min-width: 120px;
    }

    /* 文件列表优化 */
    .file-header-row {
        font-size: 11px;
        padding: 8px;
    }

    .file-header-row span:nth-child(2) {
        display: none;
    }

    .file-item {
        padding: 12px 8px;
        font-size: 12px;
    }

    .file-item span:nth-child(2) {
        display: none;
    }

    .txt-btn {
        padding: 8px;
        font-size: 11px;
        min-height: 44px;
        min-width: 44px;
    }

    /* 网格视图优化 */
    .grid-view-brutal {
        grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
        gap: 10px;
    }

    .grid-item-brutal {
        padding: 8px;
    }

    .grid-name {
        font-size: 11px;
    }

    /* 单文件操作优化 */
    .file-action-buttons {
        flex-direction: column;
    }

    .primary-action,
    .secondary-action {
        width: 100%;
        padding: 15px;
    }

    /* 文件头部优化 */
    .file-header-brutal {
        flex-direction: column;
        text-align: center;
    }

    .file-icon-box {
        margin: 0 0 10px 0;
    }

    .file-title {
        font-size: 18px;
    }

    /* 窗口头部优化 */
    .window-header {
        font-size: 14px;
        padding: 8px 12px;
    }

    .window-controls button {
        min-width: 44px;
        min-height: 44px;
    }
}

/* ====================
   预览 Overlay 核心样式 (复用)
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
    background: white;
    border: 4px solid black;
    box-shadow: 6px 6px 0px black;
    display: flex; flex-direction: column;
    animation: slam-in 0.2s cubic-bezier(0.18, 0.89, 0.32, 1.28) forwards;
}
@keyframes slam-in { 0% { transform: scale(0.9) skewX(5deg); opacity: 0; } 100% { transform: scale(1) skewX(0); opacity: 1; } }

.window-header {
    background: black; color: var(--acid-green, #ccff00);
    padding: 10px 15px; font-weight: 900; text-transform: uppercase; font-size: 18px;
    border-bottom: 4px solid black; display: flex; justify-content: space-between; align-items: center;
}
.window-controls button {
    background: var(--acid-pink, #ff00ff); border: 2px solid white; color: white;
    width: 25px; height: 25px; font-weight: bold; margin-left: 5px; box-shadow: 2px 2px 0 black;
    cursor: pointer;
}

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

/* 预览弹窗移动端优化 */
@media (max-width: 768px) {
    .preview-overlay .window-frame {
        width: 100% !important;
        max-width: calc(100vw - 20px) !important;
        min-width: unset !important;
        margin: 10px;
    }

    .brutal-video {
        max-height: 50vh;
    }

    .preview-img {
        max-height: 50vh;
    }

    .file-info-bar {
        grid-template-columns: 1fr 1fr;
    }

    .file-info-bar .file-info-item:nth-child(3) {
        display: none;
    }

    .image-stage {
        min-height: 200px;
        padding: 10px;
    }
}
</style>
