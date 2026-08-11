import { expect, test, type Page } from '@playwright/test'

async function loginAdmin(page: Page) {
  await page.goto('/manage/login')
  await page.getByRole('button', { name: '平台管理员' }).click()
  await page.getByRole('button', { name: '登录管理端' }).click()
  await page.waitForURL(/\/manage\/dashboard/)
}

test('核心页面在桌面与移动端无溢出且控制台干净', async ({
  page,
}, testInfo) => {
  const consoleIssues: string[] = []
  await loginAdmin(page)
  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') {
      consoleIssues.push(`${message.type()}: ${message.text()}`)
    }
  })
  const pages = [
    ['/manage/dashboard', '运营概览', 'dashboard'],
    ['/manage/redemption-codes', '兑换码', 'redemptions'],
    ['/manage/ai-providers', 'AI 服务', 'providers'],
    ['/manage/generation-tasks', '生成任务', 'tasks'],
    ['/manage/retouch-tickets', '人工修图工单', 'retouch'],
  ] as const

  for (const [path, title, screenshotName] of pages) {
    await page.goto(path)
    await expect(
      page.getByRole('heading', { name: title, exact: true }).last(),
    ).toBeVisible()
    await expect
      .poll(() =>
        page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth + 1,
        ),
      )
      .toBe(true)
    await page.screenshot({
      path: `/private/tmp/yingyan-admin-${testInfo.project.name}-${screenshotName}.png`,
      fullPage: true,
    })
  }

  await page.goto('/manage/generation-tasks')
  await page.getByText('自然光人像精修').first().click()
  await expect(
    page.getByRole('dialog', { name: '自然光人像精修' }),
  ).toBeVisible()
  await page.screenshot({
    path: `/private/tmp/yingyan-admin-${testInfo.project.name}-task-drawer.png`,
    fullPage: true,
  })

  await page.goto('/manage/retouch-tickets')
  await page.getByText('YY20260730-A1B2C3').click()
  await expect(
    page.getByRole('dialog', { name: 'YY20260730-A1B2C3' }),
  ).toBeVisible()
  await page.screenshot({
    path: `/private/tmp/yingyan-admin-${testInfo.project.name}-retouch-drawer.png`,
    fullPage: true,
  })

  expect(consoleIssues).toEqual([])
})
