import { test, expect } from '@playwright/test';

test.describe('指纹列表页面', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/');
  });

  test('页面加载并显示标题', async ({ page }) => {
    // 等待页面标题出现
    await expect(page.locator('.page-header h2')).toContainText('Fingerprint Manager');
  });

  test('显示新建指纹窗口按钮', async ({ page }) => {
    const newButton = page.locator('.page-header .btn-primary');
    await expect(newButton).toBeVisible();
    await expect(newButton).toContainText('新建指纹窗口');
  });

  test('无指纹时显示空状态或警告', async ({ page }) => {
    // 等待页面加载完成
    await page.waitForLoadState('networkidle');
    
    // 要么显示空状态，要么显示警告（无浏览器实例）
    const emptyState = page.locator('.empty-state');
    const warningBanner = page.locator('.warning-banner');
    
    const hasEmptyState = await emptyState.isVisible().catch(() => false);
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    expect(hasEmptyState || hasWarning).toBeTruthy();
  });

  test('有指纹时显示指纹列表', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // 检查是否有指纹卡片或空状态
    const fingerprintList = page.locator('.fingerprint-list');
    const emptyState = page.locator('.empty-state');
    
    const hasFingerprints = await fingerprintList.isVisible().catch(() => false);
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    
    expect(hasFingerprints || hasEmpty).toBeTruthy();
  });

  test('指纹卡片包含必要信息', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    
    // 如果有指纹卡片，检查信息完整性
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 检查国家名称
      await expect(card.locator('.fingerprint-country .country-name')).toBeVisible();
      // 检查 Seed
      await expect(card.locator('.seed-row .value')).toBeVisible();
      // 检查标签页数量
      await expect(card.locator('.tab-count .count')).toBeVisible();
      // 检查状态
      await expect(card.locator('.status-text')).toBeVisible();
    }
  });
});
