import { expect, test } from '@playwright/test'

test.describe('workspace smoke', () => {
  test('loads workspace shell with authenticated state', async ({ page }) => {
    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()
    await expect(page.getByTestId('service-tree-panel')).toBeVisible()
    await expect(page.getByTestId('workspace-function-renderer')).toBeVisible()
  })
})
