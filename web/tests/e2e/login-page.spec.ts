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
  })
})
