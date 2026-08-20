const path = require('path');
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests/webui',
  fullyParallel: false,
  workers: 1,
  timeout: 30_000,
  expect: { timeout: 5_000 },
  outputDir: path.join('output', 'playwright', 'results'),
  reporter: process.env.CI
    ? [['line'], ['html', { outputFolder: 'output/playwright/report', open: 'never' }]]
    : 'line',
  use: {
    baseURL: 'http://127.0.0.1:18080',
    browserName: 'chromium',
    colorScheme: 'dark',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure'
  },
  webServer: {
    command: 'sh tests/run-webui-server.sh',
    url: 'http://127.0.0.1:18080/api/health',
    reuseExistingServer: false,
    timeout: 120_000
  }
});
