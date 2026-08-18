import { expect, test } from '@playwright/test'

test.describe('login page smoke', () => {
  test('loads public login shell without authenticated setup', async ({ page }) => {
    await page.goto('/login')

    await expect(page).toHaveURL(/\/login/)
    await expect(page.getByTestId('login-page')).toBeVisible()
    await expect(page.getByTestId('login-form')).toBeVisible()
    await expect(page.getByTestId('login-username')).toBeVisible()
    await expect(page.getByTestId('login-password')).toBeVisible()
    await expect(page.getByTestId('login-submit')).toBeVisible()
    await expect(page.locator('a[href="/legal/terms"]')).toBeVisible()
    await expect(page.locator('a[href="/legal/privacy"]')).toBeVisible()
  })

  test('opens the public legal documents without authentication', async ({ page }) => {
    await page.goto('/legal/privacy')

    await expect(page).toHaveURL(/\/legal\/privacy/)
    await expect(page.getByTestId('legal-document-page')).toBeVisible()
    await expect(page.locator('a[href="/legal/terms"]')).toBeVisible()
    await expect(page.locator('a[href="/legal/privacy"]')).toBeVisible()
  })
})
