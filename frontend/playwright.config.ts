// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineConfig } from '@playwright/test'

/**
 * Smoke-e2e веб-версии (задача 1.8). Ожидает запущенный kubron-server (E2E_BASE_URL,
 * по умолчанию http://127.0.0.1:8080) с ПУСТЫМ каталогом данных (KUBRON_DATA_DIR).
 * Локально: см. docs/e2e.md. Тест кластера выполняется, только если задан E2E_KUBECONFIG.
 */
export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://127.0.0.1:8080',
    // Тесты ищут элементы по русским надписям: язык интерфейса берётся из браузера (docs/i18n.md)
    locale: 'ru-RU',
    // E2E_CHROMIUM — путь к уже установленному Chromium (если `playwright install` недоступен)
    launchOptions: process.env.E2E_CHROMIUM ? { executablePath: process.env.E2E_CHROMIUM } : {},
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
})
