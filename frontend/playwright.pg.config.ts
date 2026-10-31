// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineConfig } from '@playwright/test'

/**
 * e2e сервисного режима на PostgreSQL (gate этапа 3, задача 3.8). Ожидает kubron-server на PG с ПУСТОЙ базой
 * (E2E_PG_BASE_URL, по умолчанию http://127.0.0.1:8081). Запуск и подготовка — docs/e2e.md.
 */
export default defineConfig({
  testDir: './e2e-pg',
  timeout: 90_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0, // тесты зависят друг от друга и меняют состав команд: повтор не идемпотентен
  reporter: process.env.CI ? [['list'], ['html', { open: 'never', outputFolder: 'playwright-report-pg' }]] : 'list',
  use: {
    baseURL: process.env.E2E_PG_BASE_URL ?? 'http://127.0.0.1:8081',
    locale: 'ru-RU',
    launchOptions: process.env.E2E_CHROMIUM ? { executablePath: process.env.E2E_CHROMIUM } : {},
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
})
