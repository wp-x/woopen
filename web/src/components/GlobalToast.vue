<template>
  <div class="toast-container">
    <div 
      class="toast-box" 
      :class="{ 
        'show': uiStore.toast.show, 
        'success': uiStore.toast.type === 'success', 
        'error': uiStore.toast.type === 'error' 
      }"
    >
      <div class="toast-content">
        <div class="toast-icon">
          {{ uiStore.toast.type === 'success' ? '✅' : '⚠️' }}
        </div>
        <div class="toast-text">
          <div class="toast-title">{{ uiStore.toast.title }}</div>
          <div class="toast-desc">{{ uiStore.toast.message }}</div>
        </div>
        <button class="toast-close-btn" @click="uiStore.toast.show = false">X</button>
      </div>
      <div class="toast-progress">
        <!-- 动画逻辑：show 为 false 时 width 为 100% (准备状态)，show 为 true 时 width 变为 0% (开始倒计时) -->
        <div class="toast-bar" :style="{ transition: uiStore.toast.show ? 'width 3s linear' : 'none', width: uiStore.toast.show ? '0%' : '100%' }"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useUIStore } from '../stores/ui'

const uiStore = useUIStore()
</script>

<style scoped>
/* ====================
   顶部通知 (汉化版 - 严格复刻)
   ==================== */
.toast-container {
    position: fixed; 
    top: 20px; 
    left: 50%; 
    transform: translateX(-50%); 
    z-index: 9999; 
    pointer-events: none;
    width: auto;
}

.toast-box {
    background: var(--pure-white, #ffffff); 
    border: 4px solid var(--pure-black, #000000); 
    box-shadow: 8px 8px 0 var(--pure-black, #000000);
    min-width: 350px; 
    display: flex; 
    flex-direction: column; 
    pointer-events: auto;
    transform: translateY(-150%); 
    transition: transform 0.3s cubic-bezier(0.18, 0.89, 0.32, 1.28);
}

.toast-box.show { 
    transform: translateY(0); 
}

.toast-content { 
    padding: 15px 20px; 
    display: flex; 
    align-items: center; 
    gap: 15px; 
}

.toast-icon { 
    font-size: 30px; 
}

.toast-text { 
    display: flex; 
    flex-direction: column; 
}

.toast-title { 
    font-weight: 900; 
    font-size: 16px; 
    text-transform: uppercase; 
}

.toast-desc { 
    font-family: monospace; 
    font-size: 12px; 
    color: #555; 
    margin-top: 2px; 
}

.toast-close-btn {
    margin-left: auto; 
    background: none; 
    border: 2px solid black; 
    font-weight: bold; 
    cursor: pointer;
    padding: 2px 8px;
    font-family: monospace;
}
.toast-close-btn:hover {
    background: black;
    color: white;
}

.toast-progress { 
    height: 6px; 
    width: 100%; 
    background: black; 
    position: relative; 
}

.toast-bar { 
    height: 100%; 
    background: var(--acid-green, #ccff00); 
    /* width 和 transition 由 style 绑定动态控制 */
}

.toast-box.success .toast-bar { 
    background: var(--acid-green, #ccff00); 
}

.toast-box.error .toast-icon { 
    color: red; 
}

.toast-box.error .toast-bar {
    background: red;
}

/* 移动端响应式优化 */
@media (max-width: 768px) {
    .toast-container {
        top: calc(20px + env(safe-area-inset-top, 0px));
        left: 10px;
        right: 10px;
        transform: none;
        width: auto;
    }

    .toast-box {
        min-width: unset;
        width: 100%;
        box-shadow: 4px 4px 0 var(--pure-black, #000000);
    }

    .toast-content {
        padding: 12px 15px;
        gap: 10px;
    }

    .toast-icon {
        font-size: 24px;
    }

    .toast-title {
        font-size: 14px;
    }

    .toast-desc {
        font-size: 11px;
    }

    .toast-close-btn {
        min-width: 44px;
        min-height: 44px;
        display: flex;
        align-items: center;
        justify-content: center;
    }
}
</style>
