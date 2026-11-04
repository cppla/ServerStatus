const { test, expect } = require('@playwright/test');

const statsURL = /\/json\/stats\.json(?:\?|$)/;
const error = 'lookup "example.test": <img src=x onerror="alert(1)"> & offline';
const certificate = {
  name: 'certificate', domain: 'example.test', port: 443,
  expire_ts: 2_000_000_000, expire_days: 30, mismatch: false
};

for (const compact of [false, true]) {
  test.describe(compact ? 'SSL cards' : 'SSL table', () => {
    test.use({ viewport: compact ? { width: 390, height: 844 } : { width: 1280, height: 800 } });

    const items = page => page.locator(compact ? '#sslCards .card' : '#sslBody tr');
    const badge = item => item.locator(compact ? '.card-header .badge' : 'td:last-child .badge');
    const remaining = item => item.locator(compact ? '.kvlist > div:nth-child(3) > span:last-child' : 'td:nth-child(4)');
    const expiry = item => item.locator(compact ? '.kvlist > div:nth-child(4) > span:last-child' : 'td:nth-child(5)');
    const risk = page => page.locator('#overviewCards .overview-card', { hasText: '证书风险' }).locator('.value');

    async function openSSL(page, getCertificates) {
      await page.route(statsURL, route => route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ updated: String(Math.floor(Date.now() / 1000)), servers: [], sslcerts: getCertificates() })
      }));
      await page.goto('/');
      await page.locator('#navTabs button[data-tab="ssl"]').click();
    }

    test('errors, pending checks and expiry states agree with the overview', async ({ page }) => {
      const cases = [
        { name: 'healthy', expire_days: 8, status: '正常', cls: 'ok' },
        { name: 'failed-stale', error, mismatch: true, status: '检测失败', cls: 'err', unknown: true },
        { name: 'failed-empty', error, expire_ts: 0, expire_days: 0, status: '检测失败', cls: 'err', unknown: true },
        { name: 'pending', expire_ts: 0, expire_days: 0, status: '待检测', cls: 'warn', unknown: true },
        { name: 'expired', expire_days: 0, status: '已过期', cls: 'err' },
        { name: 'mismatch', mismatch: true, status: '域名不匹配', cls: 'err' },
        { name: 'expiring', expire_days: 7, status: '将到期', cls: 'warn' }
      ];
      await openSSL(page, () => cases.map(({ status, cls, unknown, ...fields }) => ({ ...certificate, ...fields })));
      await expect(items(page)).toHaveCount(cases.length);
      await expect(risk(page)).toHaveText('6');
      for (const entry of cases) {
        const item = items(page).filter({ hasText: entry.name });
        await expect(badge(item)).toHaveText(entry.status);
        await expect(badge(item)).toHaveClass(`badge ${entry.cls}`);
        await expect(item).toHaveClass(entry.cls === 'ok' ? (compact ? 'card' : '') :
          `${compact ? 'card ' : ''}alert-${entry.cls === 'err' ? 'critical' : 'warning'}`);
        if (entry.unknown) {
          await expect(remaining(item)).toHaveText('-');
          await expect(expiry(item)).toHaveText('-');
        }
        if (entry.error) {
          await expect(badge(item)).toHaveAttribute('title', error);
          await expect(item.locator('img')).toHaveCount(0);
        }
      }
    });

    test('polling updates success, failure and recovery without stale expiry or errors', async ({ page }) => {
      let current = { ...certificate };
      await openSSL(page, () => [current]);
      const item = items(page);
      await expect(badge(item)).toHaveText('正常');
      await expect(risk(page)).toHaveText('0');

      current = { ...certificate, error };
      await expect(badge(item)).toHaveText('检测失败');
      await expect(remaining(item)).toHaveText('-');
      await expect(expiry(item)).toHaveText('-');
      await expect(risk(page)).toHaveText('1');

      current = { ...certificate, expire_ts: 2_100_000_000, expire_days: 90 };
      await expect(badge(item)).toHaveText('正常');
      await expect(badge(item)).toHaveClass('badge ok');
      await expect(badge(item)).toHaveAttribute('title', '');
      await expect(remaining(item)).toHaveText(compact ? '90 天' : '90');
      const date = new Date(current.expire_ts * 1000).toISOString();
      await expect(expiry(item)).toHaveText(compact ? date.slice(0, 10) : date.replace('T', ' ').replace(/\.\d+Z/, ''));
      await expect(risk(page)).toHaveText('0');
      await expect(item).not.toHaveClass(/alert-/);
    });
  });
}
