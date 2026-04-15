import { expect, test, type Page } from '@playwright/test'

import { getRootPackageNode, selectWorkspaceIfNeeded } from './helpers'

async function openRootNodeActions(page: Page): Promise<string> {
  const rootNode = await getRootPackageNode(page, 'No root package node is available in the service tree.')

  await rootNode.hover()
  const nodeId = await rootNode.getAttribute('data-node-id')
  if (!nodeId) {
    throw new Error('Root node is missing data-node-id.')
  }

  await page.getByTestId(`service-tree-more-${nodeId}`).click({ force: true })
  await expect(page.getByTestId(`service-tree-action-create-directory-${nodeId}`)).toBeVisible()

  return nodeId
}

test.describe('service tree action smoke', () => {
  test('opens create directory dialog from root package actions', async ({ page }) => {
    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()

    await selectWorkspaceIfNeeded(page, 'No workspace is available for service tree interaction smoke.')
    await expect(page.getByTestId('service-tree-panel')).toBeVisible()

    const rootNodeId = await openRootNodeActions(page)
    await page.getByTestId(`service-tree-action-create-directory-${rootNodeId}`).click()

    const dialog = page.getByTestId('create-directory-dialog')
    await expect(dialog).toBeVisible()
    await expect(page.getByTestId('create-directory-name')).toBeVisible()
    await expect(page.getByTestId('create-directory-code')).toBeVisible()

    await page.getByTestId('create-directory-cancel').click()
    await expect(dialog).not.toBeVisible()
  })
})
