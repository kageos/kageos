import { expect, test, type Page } from '@playwright/test'

import { getRootPackageNode, selectWorkspaceIfNeeded } from './helpers'

async function openRootAction(page: Page, actionName: string): Promise<string> {
  const rootNode = await getRootPackageNode(page, 'No root package node is available in the service tree.')

  await rootNode.hover()
  const nodeId = await rootNode.getAttribute('data-node-id')
  if (!nodeId) throw new Error('Root node is missing data-node-id.')

  await page.getByTestId(`service-tree-more-${nodeId}`).click({ force: true })
  await expect(page.getByTestId(`service-tree-action-${actionName}-${nodeId}`)).toBeVisible()
  return nodeId
}

test.describe('workstation open smoke', () => {
  test('opens mini workstation from root package actions', async ({ page }) => {
    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()

    await selectWorkspaceIfNeeded(page, 'No workspace is available for workstation smoke.')

    const rootNodeId = await openRootAction(page, 'open-workstation')
    await page.getByTestId(`service-tree-action-open-workstation-${rootNodeId}`).click()

    const miniWorkstation = page.getByTestId('mini-workstation').first()
    await expect(miniWorkstation).toBeVisible()
    await expect(page.getByTestId('mini-workstation-header').first()).toBeVisible()
    await expect(page.getByTestId('mini-workstation-input').first()).toBeVisible()

    await page.getByTestId('mini-workstation-close').first().click()
    await expect(miniWorkstation).not.toBeVisible()
  })
})
