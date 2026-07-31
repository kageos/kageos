import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:5173'
const authStatePath = path.resolve(fileURLToPath(new URL('.', import.meta.url)), 'tests/e2e/.auth/user.json')
const emptyStorageState = { cookies: [], origins: [] }
const shouldManageLocalWebServer = /^https?:\/\/(127\.0\.0\.1|localhost):5173(?:\/|$)/.test(baseURL)

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['list'],
    ['html', { open: 'never' }],
  ],
  outputDir: 'test-results',
  globalSetup: './tests/e2e/global.setup.ts',
  use: {
    baseURL,
    storageState: authStatePath,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    testIdAttribute: 'data-testid',
  },
  projects: [
    {
      name: 'chromium',
      testIgnore: /(?:login-page|register-home)\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
      },
    },
    {
      name: 'chromium-public',
      testMatch: /(?:login-page|register-home)\.spec\.ts/,
      use: {
        ...devices['Desktop Chrome'],
        storageState: emptyStorageState,
      },
    },
  ],
  webServer: shouldManageLocalWebServer
    ? {
        command: 'npm run dev -- --host 127.0.0.1 --port 5173',
        url: baseURL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      }
    : undefined,
})
