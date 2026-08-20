import { expect, test, type Page } from '@playwright/test'
import path from 'node:path'

async function loginDemoUser(page: Page): Promise<void> {
  await page.goto('/auth')
  await page.getByLabel('邮箱').fill('demo@yingyan.local')
  await page.getByLabel('密码', { exact: true }).fill('Demo1234!')
  await page.getByRole('button', { name: '进入工作台' }).click()
}

test('auth stays inside the viewport on desktop', async ({ page }) => {
  for (const viewport of [
    { width: 1280, height: 720 },
    { width: 1440, height: 900 },
  ]) {
    await page.setViewportSize(viewport)
    await page.goto('/auth')
    await page.waitForTimeout(100)

    for (const mode of ['登录', '注册']) {
      await page.getByRole('radio', { name: mode }).click()
      const metrics = await page.evaluate(() => ({
        clientHeight: document.documentElement.clientHeight,
        scrollHeight: document.documentElement.scrollHeight,
        horizontalOverflow:
          document.documentElement.scrollWidth >
          document.documentElement.clientWidth,
      }))

      expect(metrics.scrollHeight).toBeLessThanOrEqual(metrics.clientHeight + 1)
      expect(metrics.horizontalOverflow).toBe(false)
    }
  }
})

test('register, redeem and complete a same-page creation flow', async ({
  page,
}) => {
  await page.goto('/auth')

  await page.getByRole('radio', { name: '注册' }).click()
  const uniqueEmail = `e2e-${Date.now()}@example.com`
  await page.getByLabel('邮箱').fill(uniqueEmail)
  await page.getByLabel('密码', { exact: true }).fill('E2eDemo123!')
  await page.getByLabel('确认密码').fill('E2eDemo123!')
  await page.getByText('我已阅读并同意服务协议与隐私说明').click()
  await page.getByRole('button', { name: '创建账号' }).click()

  await expect(page).toHaveURL(/\/app\/create/)
  await page.getByRole('button', { name: '兑换码' }).click()
  await page
    .getByRole('textbox', { name: '兑换码' })
    .fill('YINGYAN-START-10')
  await page.getByRole('button', { name: '兑换次数' }).click()
  await expect(page.getByRole('heading', { name: '已增加 10 次' })).toBeVisible()
  await page.getByRole('button', { name: '返回工作台' }).click()

  const sourcePath = path.resolve('public/demo/source-portrait.jpg')
  const referencePath = path.resolve('public/demo/style-coast.jpg')
  await page
    .locator('input[type="file"]')
    .first()
    .setInputFiles(sourcePath)
  await expect(page.getByText('人物原图').or(page.getByText('source-portrait.jpg'))).toBeVisible()
  await page
    .locator('input[type="file"]')
    .nth(1)
    .setInputFiles(referencePath)
  await expect(
    page.getByRole('img', { name: 'style-coast.jpg' }),
  ).toBeVisible()

  const prompt =
    '保留人物真实五官和皮肤纹理，调整成自然通透的杂志人像，清理背景杂物。'
  await page.getByLabel('你想得到什么画面').fill(prompt)
  await page.getByRole('button', { name: '优化提示词' }).click()
  await expect(page.getByText('逐项检查后确认')).toBeVisible()
  await page.getByRole('button', { name: '确认当前方案' }).click()
  await expect(page.getByText('方案已确认')).toBeVisible()

  await page.getByRole('button', { name: '生成图片' }).click()
  await expect(page.getByText(/任务正在排队|正在生成画面/)).toBeVisible()
  await expect(
    page.getByLabel('图片预览与生成结果').getByText('生成完成'),
  ).toBeVisible({ timeout: 15_000 })

  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('link', { name: '下载当前图片' }).click()
  const download = await downloadPromise
  expect(download.suggestedFilename()).toContain('映研-')

  await page.getByRole('link', { name: '任务记录' }).click()
  await expect(page.getByText(prompt.slice(0, 20))).toBeVisible()
})

test('completes a text-only generation without uploading an image', async ({
  page,
}) => {
  await loginDemoUser(page)

  await expect(page.getByRole('radio', { name: '图生图' })).toHaveCount(0)
  await expect(page.getByRole('radio', { name: '文生图' })).toHaveCount(0)

  const prompt = '傍晚海岸边的女性时尚人像，自然风吹动长发，电影感光线。'
  await expect(page.getByRole('button', { name: '生成图片' })).toBeDisabled()
  await page.getByLabel('你想得到什么画面').fill(prompt)
  await expect(page.getByRole('button', { name: '生成图片' })).toBeEnabled()
  await page.getByRole('button', { name: '生成图片' }).click()

  await expect(
    page.getByLabel('图片预览与生成结果').getByText('生成完成'),
  ).toBeVisible({ timeout: 15_000 })
})

test('workspace stays usable at the configured viewport', async (
  { page },
  testInfo,
) => {
  await loginDemoUser(page)
  await expect(page.getByRole('heading', { name: '完成一次影像创作' })).toBeVisible()
  await expect(page.getByText('影像校样条')).toBeVisible()
  await expect(page.getByText('成片校样台')).toBeVisible()
  await expect(page.getByText('需求与提示词')).toBeVisible()
  await testInfo.attach('unified-create-workspace', {
    body: await page.screenshot({ fullPage: true }),
    contentType: 'image/png',
  })
})

test('submit and complete a human retouch ticket from task records', async ({
  page,
}, testInfo) => {
  await loginDemoUser(page)
  await page.getByRole('link', { name: '任务记录' }).click()

  await page.getByText('自然光人像精修', { exact: true }).first().click()
  const taskDrawer = page.locator(
    'aside[aria-labelledby="task-detail-heading"]',
  )
  await expect(taskDrawer).toBeVisible()
  const taskDrawerLayout = await taskDrawer.evaluate((element) => ({
    viewportWidth: window.innerWidth,
    drawerWidth: element.getBoundingClientRect().width,
  }))
  if (taskDrawerLayout.viewportWidth >= 1024) {
    expect(
      taskDrawerLayout.drawerWidth / taskDrawerLayout.viewportWidth,
    ).toBeGreaterThanOrEqual(0.64)
  } else {
    expect(taskDrawerLayout.drawerWidth).toBeGreaterThanOrEqual(
      taskDrawerLayout.viewportWidth - 1,
    )
  }
  await testInfo.attach('task-detail-drawer', {
    body: await page.screenshot({ fullPage: true }),
    contentType: 'image/png',
  })
  await page.getByRole('button', { name: '申请人工精修' }).click()

  const requestDialog = page.getByRole('dialog', { name: '申请人工精修' })
  await expect(requestDialog).toBeVisible()
  await page.waitForTimeout(350)
  await testInfo.attach('retouch-request', {
    body: await page.screenshot({ fullPage: true }),
    contentType: 'image/png',
  })
  await requestDialog
    .getByLabel('人工修图要求')
    .fill('保留真实皮肤纹理，整理右侧碎发，并自然减轻眼下阴影。')
  await requestDialog
    .getByRole('button', { name: '提交人工精修需求' })
    .click()

  await expect(page.getByText('人工精修需求已提交')).toBeVisible()
  await expect(
    page.getByRole('button', { name: '查看人工修图记录' }),
  ).toBeVisible()
  await page.getByRole('button', { name: '查看人工修图记录' }).click()

  await expect(page).toHaveURL(/\/app\/retouch\/retouch_ticket_/)
  await expect(
    page.getByRole('heading', { name: '人工修图记录' }),
  ).toBeVisible()

  const ticketDrawer = page.locator(
    'aside[aria-labelledby="retouch-heading"]',
  )
  await expect(ticketDrawer).toBeVisible()
  const ticketDrawerLayout = await ticketDrawer.evaluate((element) => ({
    viewportWidth: window.innerWidth,
    drawerWidth: element.getBoundingClientRect().width,
  }))
  expect(ticketDrawerLayout.drawerWidth).toBeCloseTo(
    taskDrawerLayout.drawerWidth,
    0,
  )
  await ticketDrawer
    .getByRole('button', { name: '放大查看待精修原结果 1' })
    .click()
  const sourceLightbox = page.getByRole('dialog', {
    name: '图片预览：待精修原结果 1',
  })
  await expect(sourceLightbox).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(sourceLightbox).toBeHidden()
  await expect(ticketDrawer).toBeVisible()
  await expect(ticketDrawer.locator('.retouch-status')).toHaveText(
    '待确认报价',
    { timeout: 10_000 },
  )

  await ticketDrawer.getByRole('button', { name: /接受 3 次报价/ }).click()
  const acceptDialog = page.getByRole('dialog', {
    name: '接受人工精修报价',
  })
  await acceptDialog.getByRole('button', { name: '接受并开始处理' }).click()
  await expect(ticketDrawer.locator('.retouch-status')).toHaveText('处理中', {
    timeout: 10_000,
  })
  await expect(ticketDrawer.locator('.retouch-status')).toHaveText('待确认', {
    timeout: 10_000,
  })

  await ticketDrawer.getByRole('button', { name: '申请返修' }).click()
  const revisionDialog = page.getByRole('dialog', { name: '申请一次返修' })
  await revisionDialog
    .getByRole('textbox', { name: '返修要求' })
    .fill('再减弱一点磨皮，保留发丝边缘的自然层次。')
  await revisionDialog
    .getByRole('button', { name: '提交返修要求' })
    .click()
  await expect(ticketDrawer.locator('.retouch-status')).toHaveText('处理中')
  await expect(ticketDrawer.locator('.retouch-status')).toHaveText('待确认', {
    timeout: 10_000,
  })

  await ticketDrawer
    .getByRole('button', { name: '放大查看精修交付成片 1' })
    .click()
  const deliverableLightbox = page.getByRole('dialog', {
    name: '图片预览：人工精修成片 1',
  })
  await expect(deliverableLightbox).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(deliverableLightbox).toBeHidden()
  await expect(ticketDrawer).toBeVisible()

  const downloadPromise = page.waitForEvent('download')
  await ticketDrawer.getByRole('link', { name: '下载' }).first().click()
  await downloadPromise

  await ticketDrawer.getByRole('button', { name: '确认交付' }).click()
  const confirmDialog = page.getByRole('dialog', { name: '确认精修交付' })
  await confirmDialog.getByRole('button', { name: '确认交付完成' }).click()
  await expect(ticketDrawer.locator('.retouch-status')).toHaveText('已交付')

  let terminalListRequests = 0
  page.on('request', (request) => {
    const url = new URL(request.url())
    if (
      request.method() === 'GET' &&
      url.pathname === '/api/retouch-tickets'
    ) {
      terminalListRequests += 1
    }
  })
  await page.waitForTimeout(3_200)
  expect(terminalListRequests).toBe(0)

  for (const viewport of [
    { name: 'mobile', width: 375, height: 844 },
    { name: 'desktop', width: 1280, height: 720 },
  ]) {
    await page.setViewportSize(viewport)
    await page.waitForTimeout(100)
    const horizontalOverflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth >
        document.documentElement.clientWidth,
    )
    expect(horizontalOverflow).toBe(false)
    await testInfo.attach(`retouch-${viewport.name}`, {
      body: await page.screenshot({ fullPage: true }),
      contentType: 'image/png',
    })
  }
})

test('workspace remains coherent across responsive breakpoints', async ({
  page,
}, testInfo) => {
  await loginDemoUser(page)
  await expect(
    page.getByRole('heading', { name: '完成一次影像创作' }),
  ).toBeVisible()

  const viewports = [
    { name: 'short-mobile', width: 360, height: 640 },
    { name: 'mobile', width: 375, height: 667 },
    { name: 'tall-mobile', width: 390, height: 844 },
    { name: 'tablet', width: 768, height: 900 },
    { name: 'compact-desktop', width: 1024, height: 768 },
    { name: 'short-desktop', width: 1280, height: 720 },
    { name: 'desktop', width: 1440, height: 900 },
    { name: 'wide-desktop', width: 1920, height: 1080 },
  ]

  for (const viewport of viewports) {
    await page.setViewportSize(viewport)
    await page.waitForTimeout(100)

    const layout = await page.evaluate(() => {
      const header = document.querySelector<HTMLElement>('.app-header')
      const nav = document.querySelector<HTMLElement>('.app-header nav')
      const quote = document.querySelector<HTMLElement>('.quote-bar')
      const headerRect = header?.getBoundingClientRect()
      const navRect = nav?.getBoundingClientRect()
      const quoteRect = quote?.getBoundingClientRect()

      return {
        pageVerticalOverflow:
          document.documentElement.scrollHeight >
          document.documentElement.clientHeight + 1,
        horizontalOverflow:
          document.documentElement.scrollWidth >
          document.documentElement.clientWidth,
        mobileNavOverlapsHeader:
          window.innerWidth <= 760 &&
          Boolean(
            headerRect &&
              navRect &&
              navRect.top < headerRect.bottom,
          ),
        quoteFitsWidth: Boolean(
          quoteRect &&
            quoteRect.left >= 0 &&
            quoteRect.right <= window.innerWidth + 1,
        ),
      }
    })

    expect(layout.horizontalOverflow).toBe(false)
    expect(layout.mobileNavOverlapsHeader).toBe(false)
    expect(layout.quoteFitsWidth).toBe(true)
    if (viewport.width > 900) {
      expect(layout.pageVerticalOverflow).toBe(false)
    }

    if (viewport.width <= 760) {
      await page.locator('.quote-bar').scrollIntoViewIfNeeded()
      await page.waitForTimeout(100)
      const quoteOverlapsMobileNav = await page.evaluate(() => {
        const navRect = document
          .querySelector<HTMLElement>('.app-header nav')
          ?.getBoundingClientRect()
        const quoteRect = document
          .querySelector<HTMLElement>('.quote-bar')
          ?.getBoundingClientRect()

        return Boolean(
          navRect &&
            quoteRect &&
            quoteRect.top < navRect.bottom &&
            quoteRect.bottom > navRect.top,
        )
      })
      expect(quoteOverlapsMobileNav).toBe(false)
    }

    await testInfo.attach(`workspace-${viewport.name}`, {
      body: await page.screenshot({ fullPage: true }),
      contentType: 'image/png',
    })
  }
})
