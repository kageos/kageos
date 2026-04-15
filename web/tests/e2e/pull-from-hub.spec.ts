import { expect, test, type Page } from '@playwright/test'

import { getRootPackageNode, selectWorkspaceIfNeeded } from './helpers'

const HUB_LINK = 'hub://demo.example/workspace/sample_dir@1'
const COPIED_HUB_LINK_KEY = 'copied_hub_link'
const COPIED_DIRECTORY_KEY = 'copied_directory'

async function primeCopiedHubLink(page: Page, hubLink: string): Promise<void> {
  await page.evaluate(
    ({ key, value, directoryKey }) => {
      window.localStorage.removeItem(directoryKey)
      window.localStorage.setItem(key, value)
    },
    {
      key: COPIED_HUB_LINK_KEY,
      value: hubLink,
      directoryKey: COPIED_DIRECTORY_KEY,
    },
  )
}

async function openPullFromHubFromRootPaste(page: Page): Promise<void> {
  const rootNode = await getRootPackageNode(page, 'No root package node is available for PullFromHub smoke.')

  await rootNode.hover()
  const nodeId = await rootNode.getAttribute('data-node-id')
  if (!nodeId) {
    throw new Error('Root node is missing data-node-id.')
  }

  await page.getByTestId(`service-tree-more-${nodeId}`).click({ force: true })
  await expect(page.getByTestId(`service-tree-action-paste-${nodeId}`)).toBeVisible()
  await page.getByTestId(`service-tree-action-paste-${nodeId}`).click()
}

test.describe('pull from hub smoke', () => {
  test('opens pull dialog from root package paste action when a hub link is copied', async ({ page }) => {
    test.fixme(true, 'Clipboard-driven pull-from-hub flow is unstable in headless Chromium. Dialog hydration is covered by PullFromHubDialog.test.ts.')

    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()

    await primeCopiedHubLink(page, HUB_LINK)
    await page.reload({ waitUntil: 'domcontentloaded' })
    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()

    await selectWorkspaceIfNeeded(page, 'No workspace is available for PullFromHub smoke.')
    await openPullFromHubFromRootPaste(page)

    const dialog = page.getByTestId('pull-from-hub-dialog')
    await expect(dialog).toBeVisible()
    await expect(page.getByTestId('pull-from-hub-link')).toHaveValue(HUB_LINK)
    await expect(page.getByTestId('pull-from-hub-submit')).toBeVisible()

    await page.getByTestId('pull-from-hub-cancel').click()
    await expect(dialog).not.toBeVisible()
  })
})
