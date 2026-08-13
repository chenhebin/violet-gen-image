import { expect, test, type Page } from '@playwright/test'

async function login(page: Page, role: '平台管理员' | '修图操作员') {
  await page.goto('/manage/login')
  const credentials =
    role === '平台管理员'
      ? { email: 'admin@yingyan.local', password: 'Admin1234!' }
      : { email: 'retouch@yingyan.local', password: 'Retouch1234!' }
  await page.getByLabel('邮箱').fill(credentials.email)
  await page.getByLabel('密码').fill(credentials.password)
  await page.getByRole('button', { name: '登录管理端' }).click()
  await expect(page).toHaveURL(/\/manage\/dashboard/)
}

async function expectNoPageOverflow(page: Page) {
  await expect
    .poll(() =>
      page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth + 1,
      ),
    )
    .toBe(true)
}

test('平台管理员登录后可访问完整运营台账', async ({ page }) => {
  await login(page, '平台管理员')

  await expect(page.getByRole('heading', { name: '运营概览' })).toBeVisible()
  await expect(page.getByRole('link', { name: '兑换码', exact: true })).toHaveCount(1)
  await expect(page.getByRole('link', { name: 'AI 服务', exact: true })).toHaveCount(1)
  await expect(page.getByRole('link', { name: '人工工单', exact: true })).toHaveCount(1)
  await expectNoPageOverflow(page)
})

test('管理员可原子生成兑换码批次', async ({ page }) => {
  await login(page, '平台管理员')
  await page.goto('/manage/redemption-batches')

  await page.getByRole('button', { name: '生成兑换码' }).first().click()
  await page.locator('#batch-name').fill('E2E 咸鱼发码批次')
  await page.locator('#batch-quantity').fill('3')
  await page.locator('#batch-credits').fill('6')
  await page.getByRole('button', { name: '生成 3 个兑换码' }).click()

  const resultDialog = page.getByRole('dialog', { name: '兑换码已生成' })
  await expect(resultDialog).toBeVisible()
  await expect(
    resultDialog.getByRole('heading', {
      name: 'E2E 咸鱼发码批次',
      exact: true,
    }),
  ).toBeVisible()
  await expect(resultDialog.getByText('3 个兑换码')).toBeVisible()
})

test('管理员可修改生成批次名称', async ({ page }) => {
  await login(page, '平台管理员')
  await page.goto('/manage/redemption-batches')

  const firstRow = page.locator('tbody tr').first()
  await firstRow.getByRole('button', { name: '更多操作' }).click()
  await page.getByRole('button', { name: '修改批次名称' }).click()

  const dialog = page.getByRole('dialog', { name: '修改批次名称' })
  await expect(dialog).toBeVisible()
  await dialog.locator('#rename-batch-name').fill('E2E 已更新批次名称')
  await dialog.getByRole('button', { name: '保存名称' }).click()

  await expect(dialog).toBeHidden()
  await expect(firstRow.getByText('E2E 已更新批次名称', { exact: true })).toBeVisible()
})

test('管理员可新增服务商并切换平台生图模型', async ({ page }) => {
  await login(page, '平台管理员')
  await page.goto('/manage/ai-providers')

  await page.getByRole('button', { name: '新增服务商' }).first().click()
  await page.locator('#provider-name').fill('e2e-provider')
  await page.locator('#provider-code').fill('e2e-provider')
  await page.locator('#provider-base-url').fill('https://api.e2e.example/v1')
  await page.locator('#provider-api-key').fill('sk-e2e-provider-secret')
  await page.getByRole('button', { name: '创建服务商' }).click()
  await expect(page.getByText('e2e-provider 已创建，请先测试连接')).toBeVisible()

  await page.getByRole('button', { name: '测试连接' }).click()
  await expect(page.getByText('e2e-provider 连接正常')).toBeVisible()
  await page.getByRole('button', { name: '新增模型' }).first().click()
  await page.getByRole('button', { name: '生图模型' }).click()
  await page.locator('#model-display-name').fill('E2E Image Model')
  await page.locator('#model-id').fill('e2e-image-v1')
  await page.getByRole('button', { name: '创建模型' }).click()
  await expect(page.getByText('E2E Image Model 已创建')).toBeVisible()

  await page.getByRole('button', { name: '测试', exact: true }).click()
  await page.getByRole('button', { name: '确认并测试' }).click()
  await expect(page.getByText('E2E Image Model 能力测试通过')).toBeVisible()
  await page.getByTitle('设为平台模型').click()
  await page.getByRole('button', { name: '确认切换' }).click()
  await expect(
    page.getByText('E2E Image Model 已设为平台生图模型'),
  ).toBeVisible()
})

test('用户调次、任务与素材详情形成可追溯链路', async ({ page }) => {
  await login(page, '平台管理员')
  await page.goto('/manage/users')

  await page.getByText('anna@example.com').first().click()
  await expect(page.getByRole('heading', { name: 'anna@example.com' })).toBeVisible()
  await page.getByRole('button', { name: '调整次数' }).click()
  await page.getByRole('spinbutton', { name: '调整次数' }).fill('2')
  await page
    .getByRole('dialog', { name: '人工调整次数' })
    .getByLabel('操作原因')
    .fill('E2E 订单补偿')
  await page.getByRole('button', { name: '确认调整' }).click()
  await expect(page.getByText('用户次数已调整')).toBeVisible()

  await page.goto('/manage/generation-tasks')
  await page.getByText('自然光人像精修').first().click()
  await expect(page.getByText('需求与确认提示词')).toBeVisible()
  await expect(
    page
      .getByRole('dialog', { name: '自然光人像精修' })
      .getByText('Photon Studio Image')
      .last(),
  ).toBeVisible()

  await page.goto('/manage/assets')
  await page.getByText('人物原图.jpg').first().click()
  await page.getByRole('button', { name: '生成签名预览' }).click()
  await expect(
    page
      .getByRole('dialog', { name: '人物原图.jpg' })
      .getByText('短期签名地址已生成'),
  ).toBeVisible()
  await expectNoPageOverflow(page)
})

test('人工工单可从待评估进入待用户接受报价', async ({ page }) => {
  await login(page, '平台管理员')
  await page.goto('/manage/retouch-tickets')

  await page.getByText('YY20260730-A1B2C3').click()
  await expect(page.getByText('人工修图说明')).toBeVisible()
  await page.getByRole('button', { name: '给出报价' }).click()
  await page.getByLabel('报价次数').fill('4')
  await page.getByLabel('报价说明').fill('包含肤色、碎发与背景细节处理')
  await page.getByRole('button', { name: '发送报价' }).click()

  await expect(page.getByText('工单状态已更新')).toBeVisible()
  await expect(
    page
      .getByRole('dialog', { name: 'YY20260730-A1B2C3' })
      .getByText('待用户接受报价'),
  ).toBeVisible()
})

test('已接受报价的人工工单可开工并上传交付', async ({ page }) => {
  await login(page, '平台管理员')
  await page.goto('/manage/retouch-tickets')

  await page.getByText('YY20260728-G7H8J9').click()
  await page.getByRole('button', { name: '确认开工' }).click()
  await page
    .getByRole('dialog', { name: '确认开始处理' })
    .getByRole('button', { name: '确认开工' })
    .click()
  await expect(page.getByRole('button', { name: '上传并交付' })).toBeVisible()

  await page.getByRole('button', { name: '上传并交付' }).click()
  await page.locator('#retouch-files').setInputFiles({
    name: '人工成片.png',
    mimeType: 'image/png',
    buffer: Buffer.from('mock-image-content'),
  })
  await page
    .getByRole('dialog', { name: '交付人工成片' })
    .getByRole('button', { name: '确认交付' })
    .click()

  await expect(page.getByText('工单状态已更新')).toBeVisible()
  await expect(
    page
      .getByRole('dialog', { name: 'YY20260728-G7H8J9' })
      .getByText('待用户确认'),
  ).toBeVisible()
})

test('修图操作员只能看到工单并受到路由守卫保护', async ({ page }) => {
  await login(page, '修图操作员')

  await expect(page.getByRole('link', { name: '人工工单', exact: true })).toHaveCount(1)
  await expect(page.getByRole('link', { name: '审计日志', exact: true })).toHaveCount(0)
  await expect(page.getByRole('link', { name: '兑换码', exact: true })).toHaveCount(0)
  await expect(page.getByRole('link', { name: 'AI 服务', exact: true })).toHaveCount(0)
  await page.goto('/manage/redemption-codes')
  await expect(
    page.getByRole('heading', { name: '无权访问', exact: true }),
  ).toBeVisible()

  await page.goto('/manage/retouch-tickets')
  await expect(page.getByRole('heading', { name: '人工修图工单' })).toBeVisible()

  await page.goto('/manage/audit-logs')
  await expect(
    page.getByRole('heading', { name: '无权访问', exact: true }),
  ).toBeVisible()
  await expectNoPageOverflow(page)
})
