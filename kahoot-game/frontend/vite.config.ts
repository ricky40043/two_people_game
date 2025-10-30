import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

const basePath = process.env.VITE_BASE_PATH || '/'

// https://vitejs.dev/config/
export default defineConfig({
  base: basePath,
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    host: true,
    proxy: {
      // API 代理到後端
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      // WebSocket 代理
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue', 'vue-router', 'pinia'],
          socket: ['socket.io-client'],
        },
      },
    },
  },
  define: {
    __VUE_OPTIONS_API__: false,
    __VUE_PROD_DEVTOOLS__: false,
  },
})
