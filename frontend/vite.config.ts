// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/// <reference types="vitest/config" />
import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

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
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  // Только чистая логика (без DOM/Wails): node-окружение быстрее и не прячет случайные зависимости от window
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
  optimizeDeps: {
    include: ['monaco-editor'],
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('monaco-editor')) {
            return 'monaco';
          }
        },
      },
    },
    assetsInlineLimit: 0,
  },
})
