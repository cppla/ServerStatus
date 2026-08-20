const { test, expect } = require('@playwright/test');

const serverSpecs = [
  { name: 'alpha-online', location: 'SG-Singapore', os: 'ubuntu', online4: true },
  { name: 'bravo-online', location: 'JP-Tokyo', os: 'debian', online4: true },
  { name: 'charlie-online', location: 'US-LosAngeles', os: 'windows', online4: true },
  { name: 'delta-online', location: 'HK-HongKong', os: 'alpine', online4: true },
  { name: 'echo-offline', location: 'FR-Paris', os: 'ubuntu' },
  { name: 'foxtrot-alert', location: 'DE-Frankfurt', os: 'debian', online4: true, cpu: 95 },
  { name: 'golf-blocked', location: 'UK-London', os: 'ubuntu', online4: true, loss: 100 },
  { name: 'hotel-online', location: 'AU-Sydney', os: 'windows', online4: true },
  { name: 'india-online', location: 'CA-Toronto', os: 'debian', online4: true },
  { name: 'juliet-online', location: 'KR-Seoul', os: 'alpine', online4: true },
  { name: 'kilo-online', location: 'SG-Singapore', os: 'ubuntu', online4: true, online6: true },
  { name: 'zulu-online', location: 'JP-Osaka', os: 'debian', online4: true }
];

function serverFixture(spec, index) {
  const received = 2_000_000_000 + index * 100_000_000;
  const sent = 1_000_000_000 + index * 50_000_000;
  return {
    name: spec.name,
    type: index % 2 ? 'kvm' : 'docker',
    host: `host-${index + 1}`,
    location: spec.location,
    os: spec.os,
    online4: !!spec.online4,
    online6: !!spec.online6,
    uptime: spec.online4 || spec.online6 ? `${index + 1} 天` : '-',
    load_1: 0.05 + index / 100,
    load_5: 0.04 + index / 100,
    load_15: 0.03 + index / 100,
    cpu: spec.cpu || 12 + index,
    cpu_cores: 2 + index % 4,
    cpu_model: 'Test CPU Model',
    memory_total: 8 * 1024 * 1024,
    memory_used: (2 + index / 10) * 1024 * 1024,
    swap_total: 1024 * 1024,
    swap_used: 128 * 1024,
    hdd_total: 100 * 1024,
    hdd_used: (20 + index) * 1024,
    network_rx: 20_000 + index * 100,
    network_tx: 10_000 + index * 100,
    network_in: received,
    network_out: sent,
    last_network_in: received - 500_000_000,
    last_network_out: sent - 250_000_000,
    ping_10010: spec.loss || 1,
    ping_189: spec.loss || 2,
    ping_10086: spec.loss || 3,
    time_10010: 35 + index,
    time_189: 45 + index,
    time_10086: 55 + index,
    tcp_count: 20 + index,
    udp_count: 5 + index,
    process_count: 80 + index,
    thread_count: 160 + index,
    io_read: 100_000,
    io_write: 50_000,
    custom: 'example=35'
  };
}

const statsFixture = {
  updated: String(Math.floor(Date.now() / 1000)),
  servers: serverSpecs.map(serverFixture),
  sslcerts: []
};

test.beforeEach(async ({ page }) => {
  await page.route(/\/json\/stats\.json(?:\?|$)/, route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(statsFixture)
  }));
  await page.goto('/');
  if (page.viewportSize().width <= 700) {
    await expect(page.locator('#serversCards .card')).toHaveCount(12);
  } else {
    await expect(page.locator('#serversBody .row-server')).toHaveCount(12);
  }
});

test('filters, focused OS selector and sorting stay stable across refreshes', async ({ page }) => {
  await expect(page.locator('#serversToolbar')).toBeVisible();

  await page.getByRole('button', { name: '离线', exact: true }).click();
  await expect(page.locator('#serversBody .row-server')).toHaveCount(1);
  await expect(page.locator('#serversBody')).toContainText('echo-offline');

  await page.getByRole('button', { name: '全部', exact: true }).click();
  await page.locator('#osFilter').selectOption({ label: 'Windows' });
  await expect(page.locator('#serversBody .row-server')).toHaveCount(2);

  await page.locator('#osFilter').focus();
  const optionCount = await page.locator('#osFilter option').count();
  await page.waitForTimeout(1200);
  await expect(page.locator('#osFilter option')).toHaveCount(optionCount);
  await expect(page.locator('#osFilter')).toBeFocused();

  await page.locator('#osFilter').selectOption('all');
  await page.locator('#serverSearch').fill('KR-Seoul');
  await expect(page.locator('#serversBody .row-server')).toHaveCount(1);
  await expect(page.locator('#serversBody')).toContainText('juliet-online');

  await page.locator('#serverSearch').fill('');
  await page.locator('#sortSelect').selectOption('name');
  await page.locator('#sortDirection').click();
  await expect(page.locator('#serversBody .row-server').first().locator('.node-name')).toHaveText('zulu-online');
});

test('status filters distinguish offline and online alerts', async ({ page }) => {
  await page.getByRole('button', { name: '异常', exact: true }).click();
  await expect(page.locator('#serversBody .row-server')).toHaveCount(2);
  await expect(page.locator('#serversBody')).toContainText('foxtrot-alert');
  await expect(page.locator('#serversBody')).toContainText('golf-blocked');
  await expect(page.locator('#serversBody')).not.toContainText('echo-offline');
});

test('online node detail shows resources and labeled charts', async ({ page }) => {
  await page.locator('#serversBody .row-server', { hasText: 'alpha-online' }).click();
  await expect(page.locator('#detailModal')).toBeVisible();
  await expect(page.locator('#detailTitle')).toContainText('alpha-online');
  await expect(page.locator('#detailContent h4')).toHaveText([
    '身份', '资源', '网络', '连接', '负载趋势', '三网延迟'
  ]);
  await expect(page.locator('.chart-legend')).toContainText(['load1load5load15', '联通电信移动']);
  await expect(page.locator('.resource-meter')).toHaveCount(4);

  await page.locator('#detailClose').click();
  await expect(page.locator('#detailModal')).toBeHidden();
});

test('theme choice persists after reload', async ({ page }) => {
  await expect(page.locator('body')).not.toHaveClass(/light/);
  await page.locator('#themeToggle').click();
  await expect(page.locator('body')).toHaveClass(/light/);
  await expect.poll(() => page.evaluate(() => localStorage.getItem('theme'))).toBe('light');

  await page.reload();
  await expect(page.locator('body')).toHaveClass(/light/);
});

test('configuration page creates, updates and deletes a node through the real API', async ({ page }) => {
  await page.locator('#navTabs button[data-tab="config"]').click();
  await expect(page.locator('#adminStatus')).toContainText('管理 API 已启用');
  await page.locator('#adminToken').fill('test-token');
  await page.locator('#adminTokenForm button[type="submit"]').click();
  await expect(page.locator('#adminStatus')).toContainText('已连接管理 API');

  await page.locator('#configForm [name="username"]').fill('web-e2e');
  await page.locator('#configForm [name="name"]').fill('web-e2e-node');
  await page.locator('#configForm [name="type"]').fill('kvm');
  await page.locator('#configForm [name="host"]').fill('web-e2e-host');
  await page.locator('#configForm [name="location"]').fill('TEST');
  await page.locator('#configForm [name="password"]').fill('web-e2e-password');
  const created = page.waitForResponse(response => response.url().endsWith('/api/servers') && response.request().method() === 'POST');
  await page.locator('#configForm button[type="submit"]').click();
  await expect((await created).status()).toBe(201);
  await expect(page.locator('#configItemList')).toContainText('web-e2e-node');

  await page.locator('#configForm [name="name"]').fill('web-e2e-node-updated');
  const updated = page.waitForResponse(response => response.url().endsWith('/api/servers/web-e2e') && response.request().method() === 'PUT');
  await page.locator('#configForm button[type="submit"]').click();
  await expect((await updated).status()).toBe(200);
  await expect(page.locator('#configItemList')).toContainText('web-e2e-node-updated');

  page.once('dialog', dialog => dialog.accept());
  const deleted = page.waitForResponse(response => response.url().endsWith('/api/servers/web-e2e') && response.request().method() === 'DELETE');
  await page.locator('#deleteConfigItemBtn').click();
  await expect((await deleted).status()).toBe(200);
  await expect(page.locator('#configItemList')).not.toContainText('web-e2e-node-updated');
});

test.describe('mobile layout', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('uses cards, hides overview and keeps the detail drawer inside the viewport', async ({ page }) => {
    await expect(page.locator('#overviewCards')).toBeHidden();
    await expect(page.locator('#serversCards')).toBeVisible();
    await expect(page.locator('#serversCards .card')).toHaveCount(12);
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(391);

    await page.locator('#serversCards .card', { hasText: 'alpha-online' }).click();
    await expect(page.locator('#detailModal')).toBeVisible();
    await expect.poll(async () => {
      const drawer = await page.locator('#detailModal .detail-drawer').boundingBox();
      return Math.ceil(drawer.x + drawer.width);
    }).toBeLessThanOrEqual(391);
    const drawer = await page.locator('#detailModal .detail-drawer').boundingBox();
    expect(drawer.x).toBeGreaterThanOrEqual(0);
  });
});
