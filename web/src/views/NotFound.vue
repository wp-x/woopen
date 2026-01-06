<template>
  <div class="notfound-wrapper">
    <!-- 404 窗口 -->
    <div class="window-frame active">
      <div class="window-header" style="background: red; color: white;">
        <span>系统错误 // ERROR</span>
        <div class="window-controls">
          <button @click="$router.push('/')">X</button>
        </div>
      </div>
      
      <div class="content-area">
        <div class="error-icon">⚠️</div>
        <div class="error-code">404</div>
        <div class="error-title">// 页面未找到</div>
        <p class="terminal-text">
          &gt; 请求的资源不存在或已被删除<br>
          &gt; ERROR_CODE: NOT_FOUND<br>
          &gt; STATUS: 终端连接失败
        </p>
        
        <div class="action-row">
          <button class="btn-brutal primary" @click="$router.push('/')">
            返回首页
          </button>
          <button class="btn-brutal secondary" @click="$router.back()">
            返回上页
          </button>
        </div>
      </div>
      
      <div class="window-footer">
        WOOPEN_OS // ERROR_HANDLER_v1
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 404 页面
</script>

<style scoped>
.notfound-wrapper {
  width: 100%;
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
  background: var(--bg-color, #0a0a0a);
}

.window-frame {
  background: var(--pure-white, #fff);
  border: var(--border-thick, 4px solid #000);
  box-shadow: var(--shadow-hard, 8px 8px 0 #000);
  width: 450px;
  display: flex;
  flex-direction: column;
  animation: slam-in 0.2s cubic-bezier(0.18, 0.89, 0.32, 1.28) forwards;
}

@keyframes slam-in {
  0% { transform: scale(0.9) skewX(5deg); opacity: 0; }
  100% { transform: scale(1) skewX(0); opacity: 1; }
}

.window-header {
  background: red;
  color: white;
  padding: 10px 15px;
  font-weight: 900;
  text-transform: uppercase;
  font-size: 18px;
  border-bottom: var(--border-thick, 4px solid #000);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.window-controls button {
  background: var(--acid-pink, #ff00ff);
  border: 2px solid white;
  color: white;
  width: 25px;
  height: 25px;
  font-weight: bold;
  cursor: pointer;
  box-shadow: 2px 2px 0 black;
}

.content-area {
  padding: 30px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.error-icon {
  font-size: 60px;
  margin-bottom: 10px;
  animation: shake 0.5s infinite;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}

.error-code {
  font-size: 80px;
  font-weight: 900;
  color: red;
  line-height: 1;
  text-shadow: 4px 4px 0 var(--pure-black, #000);
  animation: glitch 0.3s infinite;
}

@keyframes glitch {
  0% { text-shadow: 4px 4px 0 #000; }
  25% { text-shadow: -4px 4px 0 var(--acid-cyan, #0ff); }
  50% { text-shadow: 4px -4px 0 var(--acid-pink, #f0f); }
  75% { text-shadow: -4px -4px 0 var(--acid-green, #cf0); }
  100% { text-shadow: 4px 4px 0 #000; }
}

.error-title {
  font-size: 20px;
  font-weight: 900;
  text-transform: uppercase;
  margin: 10px 0;
}

.terminal-text {
  font-family: monospace;
  font-size: 12px;
  color: #555;
  text-align: left;
  background: #f5f5f5;
  padding: 15px;
  border: 2px dashed #000;
  width: 100%;
  margin: 15px 0;
}

.action-row {
  display: flex;
  gap: 10px;
  width: 100%;
}

.btn-brutal {
  flex: 1;
  border: none;
  padding: 15px 20px;
  font-size: 14px;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 1px;
  cursor: pointer;
  transition: transform 0.1s;
}

.btn-brutal:hover {
  box-shadow: 4px 4px 0 var(--pure-black, #000);
  transform: translate(-2px, -2px);
}

.btn-brutal:active {
  transform: translate(0, 0);
  box-shadow: none;
}

.btn-brutal.primary {
  background: var(--acid-green, #cf0);
  color: var(--pure-black, #000);
}

.btn-brutal.secondary {
  background: var(--pure-black, #000);
  color: var(--pure-white, #fff);
}

.btn-brutal.secondary:hover {
  background: var(--acid-pink, #f0f);
  color: var(--pure-black, #000);
}

.window-footer {
  padding: 10px;
  background: #eee;
  font-family: monospace;
  text-align: center;
  font-size: 10px;
  border-top: var(--border-thick, 4px solid #000);
}

/* Mobile */
@media (max-width: 500px) {
  .window-frame {
    width: 100%;
  }
  
  .error-code {
    font-size: 60px;
  }
  
  .action-row {
    flex-direction: column;
  }
}
</style>
