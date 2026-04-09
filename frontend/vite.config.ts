import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    {
      name: 'wails-stub',
      resolveId(id) {
        if (id.startsWith('wailsjs/')) {
          return id
        }
      },
      load(id) {
        if (id.startsWith('wailsjs/')) {
          return 'export default {}' // Stub for Vite
        }
      }
    }
  ],
  optimizeDeps: {
    include: ['monaco-editor'],
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'monaco': ['monaco-editor'],
        },
      },
    },
    assetsInlineLimit: 0,
  },
})
