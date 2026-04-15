import { expect, test, type Locator, type Page } from '@playwright/test'

const DEFAULT_WAIT_MS = 10_000
const WORKSPACE_CARD_SELECTOR =
  '[data-testid^="workspace-card-"], [data-testid^="workspace-card-public-"], [data-testid^="workspace-card-system-"]'
const ROOT_PACKAGE_SELECTOR = '[data-node-type="package"][data-root-node="true"]'

export async function selectWorkspaceIfNeeded(page: Page, reason: string): Promise<void> {
  const dialog = page.getByTestId('workspace-list-dialog')
  if (!(await dialog.isVisible().catch(() => false))) return

  const workspaceCards = page.locator(WORKSPACE_CARD_SELECTOR)
  await workspaceCards
    .first()
    .waitFor({ state: 'visible', timeout: DEFAULT_WAIT_MS })
    .catch(() => null)

  const cardCount = await workspaceCards.count()
  test.skip(cardCount === 0, reason)

  await workspaceCards.first().click()
  await expect(dialog).not.toBeVisible()
}

export async function getRootPackageNode(page: Page, reason: string): Promise<Locator> {
  const rootNode = page.locator(ROOT_PACKAGE_SELECTOR).first()
  await rootNode.waitFor({ state: 'visible', timeout: DEFAULT_WAIT_MS }).catch(() => null)

  const rootCount = await rootNode.count()
  test.skip(rootCount === 0, reason)

  return rootNode
}
