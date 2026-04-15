import { expect, test, type Locator, type Page } from '@playwright/test'

import { getRootPackageNode, selectWorkspaceIfNeeded } from './helpers'

async function openRootActions(page: Page): Promise<string> {
  const rootNode = await getRootPackageNode(page, 'No root package node is available in the service tree.')

  await rootNode.hover()
  const nodeId = await rootNode.getAttribute('data-node-id')
  if (!nodeId) throw new Error('Root node is missing data-node-id.')

  await page.getByTestId(`service-tree-more-${nodeId}`).click({ force: true })
  return nodeId
}

function visibleAction(...actions: Locator[]): Promise<Locator | null> {
  return (async () => {
    for (const action of actions) {
      if (await action.isVisible().catch(() => false)) return action
    }
    return null
  })()
}

test.describe('hub dialog smoke', () => {
  test('opens publish or push dialog from root package actions', async ({ page }) => {
    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    await expect(page.getByTestId('workspace-view')).toBeVisible()

    await selectWorkspaceIfNeeded(page, 'No workspace is available for hub dialog smoke.')

    const rootNodeId = await openRootActions(page)
    const publishAction = page.getByTestId(`service-tree-action-publish-to-hub-${rootNodeId}`)
    const pushAction = page.getByTestId(`service-tree-action-push-to-hub-${rootNodeId}`)
    const action = await visibleAction(publishAction, pushAction)

    test.skip(!action, 'No publish or push action is available on the root package node.')

    await action!.click()

    const publishDialog = page.getByTestId('publish-to-hub-dialog')
    const pushDialog = page.getByTestId('push-to-hub-dialog')
    const activeDialog = (await publishDialog.isVisible().catch(() => false)) ? publishDialog : pushDialog

    await expect(activeDialog).toBeVisible()
    await expect(
      publishDialog === activeDialog
        ? page.getByTestId('publish-to-hub-submit')
        : page.getByTestId('push-to-hub-submit'),
    ).toBeVisible()
  })
})
