// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/// <reference types="vitest/config" />
import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const isWeb = mode === 'web'
  // Веб-сборка и юнит-тесты не используют Wails: его runtime подменяется пустышкой, чтобы они не
  // зависели от `wails generate module` (типы для vue-tsc он по-прежнему требует).
  // В вебе ветки с вызовом биндингов дополнительно вырезаются по VITE_TARGET.
  const stubWails = isWeb || mode === 'test' || !!process.env.VITEST

  return {
    plugins: [
      vue(),
      ...(isWeb ? [{
        // Метка веб-сборки: kubron-server отказывается раздавать десктопный frontend/dist
        name: 'web-build-marker',
        transformIndexHtml: () => [{ tag: 'meta', attrs: { name: 'kubron-target', content: 'web' }, injectTo: 'head' as const }],
      }] : []),
      ...(stubWails ? [{
        name: 'wails-runtime-stub',
        enforce: 'pre' as const,
        resolveId(id: string) {
          if (/wailsjs\/(runtime|go\/main)\//.test(id)) return '\0wails-stub'
        },
        load(id: string) {
          if (id === '\0wails-stub') return 'export const EventsOn = () => () => {}\nexport const BrowserOpenURL = () => {}\nexport default {}'
        },
      }] : []),
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
    // Только чистая логика (без DOM/Wails): node-окружение быстрее и не прячет случайные зависимости от window.
    // Wails-runtime подменяется плагином wails-runtime-stub, поэтому `npm test` не требует `wails generate module`.
    test: {
      environment: 'node',
      include: ['src/**/*.test.ts'],
    },
    // Разработка веб-версии: `npm run dev:web` + запущенный kubron-server на 127.0.0.1:8080.
    // Host и Origin остаются адресом dev-сервера, поэтому проверка Origin на бэкенде проходит.
    server: isWeb ? { proxy: { '/api': { target: 'http://127.0.0.1:8080', ws: true } } } : undefined,
    optimizeDeps: {
      include: ['monaco-editor'],
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks(id: string) {
            if (id.includes('monaco-editor')) {
              return 'monaco';
            }
          },
        },
      },
      assetsInlineLimit: 0,
    },
  }
})