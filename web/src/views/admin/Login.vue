<template>
  <div class="login-wrapper">
    <!-- Login Window -->
    <div id="view-login" class="window-frame active">
      <div class="window-header">
        <span class="header-title">{{ siteConfig.login_title }}</span>
        <div class="window-controls">
          <button @click="playSound('click')">X</button>
        </div>
      </div>
      
      <div class="login-grid">
        <!-- Left: ID Card Section -->
        <div class="id-card-section">
          <div class="big-bg-text">USER</div>
          <div class="avatar-frame">
            <img v-if="isAvatarUrl" :src="siteConfig.login_avatar" alt="Avatar" class="avatar-img" />
            <span v-else>{{ siteConfig.login_avatar }}</span>
          </div>
          <div class="tags-container">
            <div class="tag">{{ siteConfig.login_role_tag }}</div>
            <div class="tag level-tag">{{ siteConfig.login_level_tag }}</div>
          </div>
          <div class="user-title" v-html="formattedSystemName"></div>
          <p class="terminal-text">
            // 终端接入中...<br>
            // 加密节点已就绪<br>
            // 请勿切断电源或关闭
          </p>
        </div>

        <!-- Right: Form Section -->
        <div class="login-form-section">
          <h2 class="form-title">身份验证程序</h2>
          
          <label class="brutal-label">> 用户名</label>
          <input type="text" :value="siteConfig.login_username" readonly class="brutal-input readonly">
          
          <label class="brutal-label">> 授权时长</label>
          <div class="expire-options">
            <label v-for="opt in expireOptions" :key="opt.value" class="expire-option"
                   :class="{ active: expire === opt.value }" @click="expire = opt.value; playSound('click')">
              {{ opt.label }}
            </label>
          </div>

          <label class="brutal-label">> 访问密码</label>
          <input 
            type="password" 
            v-model="password" 
            placeholder="_" 
            class="brutal-input sound-type"
            @keyup.enter="handleLogin"
            @input="handleInput"
            ref="passwordInput"
          >
          
          <button 
            class="btn-brutal sound-click" 
            ref="loginBtn"
            @click="handleLogin"
            @mousedown="playSound('click')"
            @touchstart="playSound('click')"
          >
            启动系统
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { authApi, publicApi } from '../../api'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const password = ref('')
const expire = ref('30d')
const expireOptions = [
  { value: '7d', label: '7天' },
  { value: '30d', label: '30天' },
  { value: 'forever', label: '永久' }
]
const loading = ref(false)
const passwordInput = ref<HTMLInputElement | null>(null)
const loginBtn = ref<HTMLButtonElement | null>(null)

// 站点配置
const siteConfig = ref({
  login_username: 'Administrator',
  login_title: 'WoOpen_Auth_v10.exe',
  login_avatar: '💀',
  login_role_tag: 'ROLE: ADMIN',
  login_level_tag: 'LEVEL: 99',
  login_system_name: 'WOOPEN CLOUD SYSTEM'
})

// 检测头像是否为 URL
const isAvatarUrl = computed(() => {
  const avatar = siteConfig.value.login_avatar
  return avatar && (avatar.startsWith('http://') || avatar.startsWith('https://') || avatar.startsWith('/'))
})

// 格式化系统名称（用空格分隔换行）
const formattedSystemName = computed(() => {
  const name = siteConfig.value.login_system_name || 'WOOPEN CLOUD SYSTEM'
  return name.split(' ').join('<br>')
})

// 加载站点配置
async function loadSiteConfig() {
  try {
    const res = await publicApi.getSiteConfig()
    if (res.code === 0 && res.data) {
      siteConfig.value = {
        login_username: res.data.login_username || 'Administrator',
        login_title: res.data.login_title || 'WoOpen_Auth_v10.exe',
        login_avatar: res.data.login_avatar || '💀',
        login_role_tag: res.data.login_role_tag || 'ROLE: ADMIN',
        login_level_tag: res.data.login_level_tag || 'LEVEL: 99',
        login_system_name: res.data.login_system_name || 'WOOPEN CLOUD SYSTEM'
      }
    }
  } catch (error) {
    // 加载失败使用默认值
    console.log('加载站点配置失败，使用默认值')
  }
}

// Audio Logic from reference
const AudioContext = window.AudioContext || (window as any).webkitAudioContext
let ctx: AudioContext

function initAudio() {
  if (!ctx) {
    ctx = new AudioContext()
  }
}

function playSound(type: 'click' | 'type') {
    initAudio()
    if(ctx.state === 'suspended') ctx.resume()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    
    osc.connect(gain)
    gain.connect(ctx.destination)

    const now = ctx.currentTime

    if (type === 'click') {
        osc.type = 'square'
        osc.frequency.setValueAtTime(150, now)
        osc.frequency.exponentialRampToValueAtTime(40, now + 0.1)
        gain.gain.setValueAtTime(0.3, now)
        gain.gain.linearRampToValueAtTime(0, now + 0.1)
        osc.start(now)
        osc.stop(now + 0.1)
    } else if (type === 'type') {
        osc.type = 'sawtooth'
        osc.frequency.setValueAtTime(800, now)
        gain.gain.setValueAtTime(0.05, now)
        gain.gain.linearRampToValueAtTime(0, now + 0.05)
        osc.start(now)
        osc.stop(now + 0.05)
    }
}

function handleInput() {
    playSound('type')
}

async function handleLogin() {
  if (!password.value) {
    // Custom error style
    alert('访问拒绝: 需要输入密码') 
    return
  }

  loading.value = true
  
  // Animation Sequence
  if (loginBtn.value) {
      const btn = loginBtn.value
      const originalText = "启动系统"
      btn.innerText = "正在验证..."
      btn.style.background = "var(--acid-cyan)"
      btn.style.color = "var(--pure-black)"

      let i = 0
      const flickerInterval = setInterval(() => {
        i++
        if(i % 2 === 0) btn.style.background = "var(--pure-white)"
        else btn.style.background = "var(--acid-cyan)"
      }, 100)

      try {
        const res = await authApi.login(password.value, expire.value)
        authStore.setToken(res.data.token)
        
        clearInterval(flickerInterval)
        btn.innerText = "权限已确认"
        btn.style.background = "var(--acid-green)"
        
        setTimeout(() => {
            router.push('/admin/dashboard')
        }, 500)
        
      } catch (error) {
        clearInterval(flickerInterval)
        btn.style.background = "red"
        btn.style.color = "white"
        btn.innerText = "访问被拒绝"
        
        // Reset button after delay
        setTimeout(() => {
            if(btn) {
                btn.style.background = "" // clear inline style to revert to css
                btn.style.color = ""
                btn.innerText = originalText
            }
            password.value = ''
        }, 1500)
      } finally {
          loading.value = false
      }
  }
}

onMounted(() => {
    loadSiteConfig()
    passwordInput.value?.focus()
})
</script>

<style scoped>
/* Scoped styles specific to Login layout structure */
.login-wrapper {
    width: 100%;
    height: 100vh;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 20px;
}

#view-login {
    width: 800px;
    min-height: 450px;
    background: white;
}

.login-grid {
    display: grid;
    grid-template-columns: 1fr 1.2fr;
    height: 100%;
    overflow: hidden;
}

/* Header Tweaks */
.header-title {
    font-family: inherit;
}

/* Left Section */
.id-card-section {
    background: var(--acid-green);
    border-right: var(--border-thick);
    padding: 30px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    position: relative;
    overflow: hidden;
}

.big-bg-text {
    position: absolute;
    font-size: 150px;
    font-weight: 900;
    color: rgba(0,0,0,0.05);
    transform: rotate(-90deg);
    right: -60px;
    bottom: 0;
    pointer-events: none;
}

.avatar-frame {
    width: 120px;
    height: 120px;
    background: var(--pure-white);
    border: var(--border-thick);
    margin-bottom: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 50px;
    box-shadow: 8px 8px 0 var(--pure-black);
}

.avatar-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.tags-container {
    margin-bottom: 15px;
}

.level-tag {
    background: var(--acid-pink);
    color: white;
}

.user-title {
    font-size: 40px;
    line-height: 0.9;
    font-weight: 900;
    margin-bottom: 10px;
    text-transform: uppercase;
}

.terminal-text {
    font-family: monospace; 
    font-size: 12px; 
    margin-top: 10px;
}

/* Right Section */
.login-form-section {
    padding: 40px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    background: white;
}

.form-title {
    font-size: 30px; 
    margin: 0 0 20px 0; 
    text-transform: uppercase;
}

.expire-options {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
}

.expire-option {
    flex: 1;
    text-align: center;
    padding: 8px 0;
    border: var(--border-thick);
    background: var(--pure-white);
    cursor: pointer;
    font-weight: 700;
    user-select: none;
}

.expire-option.active {
    background: var(--acid-green);
    box-shadow: 4px 4px 0 var(--pure-black);
}

.readonly {
    cursor: not-allowed;
    background: #e0e0e0;
    color: #555;
    border-style: dashed;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
    .login-wrapper {
        display: block; /* Allow scrolling */
        height: auto;
        padding: 10px;
        padding-top: 40px;
    }

    #view-login {
        width: 100%;
        height: auto;
        min-height: auto;
    }

    .login-grid {
        display: flex;
        flex-direction: column;
        grid-template-columns: none;
    }
    
    .id-card-section {
        border-right: none;
        border-bottom: var(--border-thick);
        padding: 20px;
        align-items: center;
        text-align: center;
    }
    
    .big-bg-text {
        display: none;
    }
    
    .avatar-frame {
        width: 80px;
        height: 80px;
        font-size: 40px;
        box-shadow: 4px 4px 0 var(--pure-black);
    }
    
    .user-title {
        font-size: 24px;
        margin-bottom: 15px;
    }
    
    .login-form-section {
        padding: 20px;
    }

    .form-title {
        font-size: 20px;
    }
}
</style>
