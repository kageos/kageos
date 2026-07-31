import { expect, test, type Page } from '@playwright/test'

const username = 'new_home_user'
const password = 'secret123'

async function fillRegistrationForm(page: Page) {
  await page.goto('/register')
  await expect(page.getByTestId('register-page')).toBeVisible()
  await page.getByTestId('register-username').fill(username)
  await page.getByTestId('register-email').fill('new-home@example.com')
  await page.getByTestId('register-password').fill(password)
  await page.getByTestId('register-company-name').fill('New Home Company')
  await page.getByTestId('register-company-code').fill('new_home_company')
  await page.getByTestId('register-code').fill('123456')
}

test.describe('email registration personal workspace', () => {
  test('logs in and enters the new private Home after registration', async ({ page }) => {
    const registrationRequests: Record<string, unknown>[] = []
    const loginRequests: Record<string, unknown>[] = []
    let bootstrapToken = ''

    await page.route('**/workspace/api/**', async route => {
      await route.abort()
    })
    await page.route('**/workspace/api/v1/app/personal-workspace', async route => {
      bootstrapToken = route.request().headers()['x-token'] || ''
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            created: true,
            app: {
              id: 1,
              user: username,
              code: 'home',
              name: '我的空间',
              status: 'enabled',
              version: 'v1',
              nats_id: 1,
              host_id: 1,
              is_public: false,
              is_personal_workspace: true,
              hide_unauthorized_nodes: false,
              admins: '',
              type: 0,
              created_at: '2026-07-30 12:00:00',
              updated_at: '2026-07-30 12:00:00',
            },
          },
          msg: '成功',
        }),
      })
    })
    await page.route('**/hr/api/v1/auth/register', async route => {
      registrationRequests.push(route.request().postDataJSON())
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: { user_id: 42 },
          msg: '成功',
        }),
      })
    })
    await page.route('**/hr/api/v1/auth/login', async route => {
      loginRequests.push(route.request().postDataJSON())
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: {
            token: 'new-user-access-token',
            refresh_token: 'new-user-refresh-token',
            user: {
              id: 42,
              username,
              email: 'new-home@example.com',
              company_code: 'new_home_company',
              register_type: 'email',
              avatar: '',
              email_verified: true,
              status: 'active',
              created_at: '2026-07-30T12:00:00+08:00',
            },
          },
          msg: '成功',
        }),
      })
    })

    await fillRegistrationForm(page)
    await page.getByTestId('register-submit').click()

    await expect(page).toHaveURL(new RegExp(`/workspace/${username}/home$`))
    expect(registrationRequests).toHaveLength(1)
    expect(registrationRequests[0]).toMatchObject({
      username,
      password,
      email: 'new-home@example.com',
      code: '123456',
      company_action: 'create',
      company_code: 'new_home_company',
      company_name: 'New Home Company',
    })
    expect(loginRequests).toEqual([{ username, password }])
    expect(bootstrapToken).toBe('new-user-access-token')
  })

  test('falls back to login when automatic login is unavailable', async ({ page }) => {
    let registerCalls = 0
    let loginCalls = 0

    await page.route('**/hr/api/v1/auth/register', async route => {
      registerCalls += 1
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          data: { user_id: 42 },
          msg: '成功',
        }),
      })
    })
    await page.route('**/hr/api/v1/auth/login', async route => {
      loginCalls += 1
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 7,
          data: {},
          msg: '自动登录暂不可用，请手动登录',
        }),
      })
    })

    await fillRegistrationForm(page)
    await page.getByTestId('register-submit').click()

    await expect(page).toHaveURL(/\/login$/)
    expect(registerCalls).toBe(1)
    expect(loginCalls).toBe(1)
  })
})
