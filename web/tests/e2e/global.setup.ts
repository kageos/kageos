import fs from 'node:fs/promises'
import path from 'node:path'

import type { FullConfig, Page } from '@playwright/test'
import { chromium } from '@playwright/test'

type InjectedUser = Record<string, unknown> | null

const AUTH_STATE_PATH = path.resolve(process.cwd(), 'tests/e2e/.auth/user.json')
const DEFAULT_BASE_URL = 'http://127.0.0.1:5173'
const DEFAULT_TIMEOUT_MS = 30_000

function shouldSkipAuthSetup(): boolean {
  return process.env.E2E_SKIP_AUTH_SETUP === '1'
}

function getBaseURL(config: FullConfig): string {
  return config.projects[0]?.use?.baseURL?.toString() || process.env.PLAYWRIGHT_BASE_URL || DEFAULT_BASE_URL
}

function getTimeoutMs(): number {
  const rawValue = process.env.PLAYWRIGHT_LOGIN_TIMEOUT_MS
  if (!rawValue) return DEFAULT_TIMEOUT_MS

  const parsedValue = Number.parseInt(rawValue, 10)
  return Number.isFinite(parsedValue) && parsedValue > 0 ? parsedValue : DEFAULT_TIMEOUT_MS
}

function getInjectedUser(): InjectedUser {
  const userJson = process.env.E2E_USER_JSON?.trim()
  if (userJson) {
    try {
      return JSON.parse(userJson) as Record<string, unknown>
    } catch (error) {
      throw new Error(`E2E_USER_JSON 不是合法 JSON: ${String(error)}`)
    }
  }

  const username = process.env.E2E_USERNAME?.trim()
  if (!username) return null

  return { username }
}

async function ensureAuthDirectory(): Promise<void> {
  await fs.mkdir(path.dirname(AUTH_STATE_PATH), { recursive: true })
}

async function assertBaseURLReachable(baseURL: string): Promise<void> {
  try {
    const response = await fetch(baseURL, { redirect: 'manual' })
    if (response.status >= 500) {
      throw new Error(`HTTP ${response.status}`)
    }
  } catch (error) {
    throw new Error(
      [
        `无法访问 Playwright 基础地址: ${baseURL}`,
        '请确认前端可访问。若使用本地开发链路，可直接在 web/ 下运行 `npm run dev -- --host 127.0.0.1 --port 5173`。',
        '若只跑前端连线上后端，请在 `web/.env.development.local` 中配置 `VITE_PROXY_TARGET` 后再重试。',
        `原始错误: ${String(error)}`,
      ].join('\n'),
    )
  }
}

async function assertAuthenticated(page: Page, timeoutMs: number): Promise<void> {
  await page.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: timeoutMs })
  await page.getByTestId('workspace-view').waitFor({ state: 'visible', timeout: timeoutMs })
}

export default async function globalSetup(config: FullConfig): Promise<void> {
  await ensureAuthDirectory()

  const baseURL = getBaseURL(config)
  const timeoutMs = getTimeoutMs()

  await assertBaseURLReachable(baseURL)

  if (shouldSkipAuthSetup()) {
    return
  }

  const injectedToken = process.env.E2E_TOKEN?.trim()
  const refreshToken = process.env.E2E_REFRESH_TOKEN?.trim()
  const username = process.env.E2E_USERNAME?.trim()
  const password = process.env.E2E_PASSWORD?.trim()

  if (!injectedToken && (!username || !password)) {
    throw new Error(
      '缺少 E2E 登录信息。请提供 E2E_USERNAME + E2E_PASSWORD，或提供 E2E_TOKEN（可选 E2E_REFRESH_TOKEN / E2E_USER_JSON）。',
    )
  }

  const browser = await chromium.launch()

  try {
    const context = await browser.newContext({ baseURL })

    if (injectedToken) {
      const user = getInjectedUser()

      await context.addInitScript(
        ({ token, refreshTokenValue, userValue }) => {
          window.localStorage.setItem('token', token)
          if (refreshTokenValue) {
            window.localStorage.setItem('refresh_token', refreshTokenValue)
          }
          if (userValue) {
            window.localStorage.setItem('user', JSON.stringify(userValue))
          }
        },
        {
          token: injectedToken,
          refreshTokenValue: refreshToken || '',
          userValue: user,
        },
      )

      const page = await context.newPage()
      await page.goto('/', { waitUntil: 'domcontentloaded', timeout: timeoutMs })
      try {
        await assertAuthenticated(page, timeoutMs)
      } catch (error) {
        throw new Error(
          [
            '基于 token 的登录态注入后，页面仍未进入工作区。',
            '请检查 token/refresh_token 是否有效，以及前端代理目标是否可访问。',
            `原始错误: ${String(error)}`,
          ].join('\n'),
        )
      }
    } else {
      const page = await context.newPage()
      await page.goto('/login', { waitUntil: 'domcontentloaded', timeout: timeoutMs })
      await page.getByTestId('login-username').fill(username!)
      await page.getByTestId('login-password').fill(password!)
      await page.getByTestId('login-submit').click()
      try {
        await assertAuthenticated(page, timeoutMs)
      } catch (error) {
        throw new Error(
          [
            '真实登录后未能进入工作区。',
            '请检查账号密码、后端网关状态，以及 `web/.env.development.local` 中的代理配置。',
            `原始错误: ${String(error)}`,
          ].join('\n'),
        )
      }
    }

    await context.storageState({ path: AUTH_STATE_PATH })
  } finally {
    await browser.close()
  }
}
