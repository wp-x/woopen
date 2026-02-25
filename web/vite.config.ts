import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://localhost:10010',
        changeOrigin: true
      },
      '/dav': {
        target: 'http://localhost:10010',
        changeOrigin: true
      },
      '/s/': {
        target: 'http://localhost:10010',
        changeOrigin: true,
        bypass: (req) => {
          const accept = req.headers?.accept || ''
          if (!accept.includes('text/html')) {
            return
          }
          const url = req.url || ''
          const path = url.split('?')[0]
          // 仅对分享页 /s/:code 生效，避免下载/预览等请求被 SPA 吞掉
          if (/^\/s\/[^/]+\/?$/.test(path)) {
            return '/index.html'
          }
        }
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: false
  }
})
