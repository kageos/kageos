import { expect, test, type Page } from '@playwright/test'

import { getRootPackageNode, selectWorkspaceIfNeeded } from './helpers'

async function openRootWorkstation(page: Page): Promise<void> {
  const rootNode = await getRootPackageNode(page, 'No root package node is available in the service tree.')

  await rootNode.hover()
  const nodeId = await rootNode.getAttribute('data-node-id')
  if (!nodeId) throw new Error('Root node is missing data-node-id.')

  await page.getByTestId(`service-tree-more-${nodeId}`).click({ force: true })
  await page.getByTestId(`service-tree-action-open-workstation-${nodeId}`).click()
}

test.describe('workstation composer smoke', () => {
  test('enables send button after typing into mini workstation', async ({ page }) => {
    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()

    await selectWorkspaceIfNeeded(page, 'No workspace is available for workstation composer smoke.')
    await openRootWorkstation(page)

    const input = page.getByTestId('mini-workstation-input').first()
    const sendButton = page.getByTestId('mini-workstation-send').first()

    await expect(input).toBeVisible()
    await expect(sendButton).toBeDisabled()

    await input.fill('请帮我分析一下当前目录')
    await expect(sendButton).toBeEnabled()
  })
})
