import { expect, test, type Locator, type Page } from '@playwright/test'

async function ensureWorkspaceListDialog(page: Page): Promise<Locator> {
  const dialog = page.getByTestId('workspace-list-dialog')

  if (await dialog.isVisible().catch(() => false)) {
    return dialog
  }

  const currentSwitcher = page.getByTestId('app-switcher-current')
  const emptySwitcher = page.getByTestId('app-switcher-empty')
  const switcher = (await currentSwitcher.isVisible().catch(() => false)) ? currentSwitcher : emptySwitcher
  await switcher.click()
  await expect(dialog).toBeVisible()
  return dialog
}

test.describe('workspace navigation smoke', () => {
  test('opens workspace chooser and create workspace dialog', async ({ page }) => {
    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()

    const dialog = await ensureWorkspaceListDialog(page)
    await expect(dialog.getByTestId('workspace-list-tabs')).toBeVisible()

    await page.getByRole('tab', { name: /全部工作空间|All Workspaces/i }).click()
    await page.getByRole('tab', { name: /系统工作空间|System Workspaces/i }).click()
    await page.getByRole('tab', { name: /我的工作空间|My Workspaces/i }).click()

    await page.getByTestId('workspace-list-create').click()

    const createDialog = page.getByTestId('create-app-dialog')
    await expect(createDialog).toBeVisible()
    await expect(page.getByTestId('create-app-name')).toBeVisible()
    await expect(page.getByTestId('create-app-code')).toBeVisible()

    await page.getByTestId('create-app-cancel').click()
    await expect(createDialog).not.toBeVisible()
  })
})
