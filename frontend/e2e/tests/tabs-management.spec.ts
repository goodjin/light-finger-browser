import { test, expect } from '@playwright/test';

test.describe('标签页管理', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.waitForLoadState('networkidle');
  });

  test('展开/折叠指纹卡片', async ({ page }) => {
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 初始状态 - 展开区域不可见
      const tabsSection = card.locator('.fingerprint-tabs-section');
      await expect(tabsSection).not.toBeVisible();
      
      // 点击卡片展开
      await card.locator('.fingerprint-row').click();
      await expect(tabsSection).toBeVisible();
      
      // 点击卡片折叠
      await card.locator('.fingerprint-row').click();
      await expect(tabsSection).not.toBeVisible();
    }
  });

  test('展开区域显示标签页列表', async ({ page }) => {
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 展开卡片
      await card.locator('.fingerprint-row').click();
      
      // 检查标签页列表区域
      const tabsSection = card.locator('.fingerprint-tabs-section');
      await expect(tabsSection).toBeVisible();
      
      // 检查标签页列表头部
      await expect(tabsSection.locator('.tabs-header')).toBeVisible();
      await expect(tabsSection.locator('.tabs-header span')).toContainText('标签页列表');
    }
  });

  test('新建标签页按钮存在', async ({ page }) => {
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 展开卡片
      await card.locator('.fingerprint-row').click();
      
      // 检查新建标签页按钮
      const newTabBtn = card.locator('.btn-icon.btn-new-tab');
      await expect(newTabBtn).toBeVisible();
    }
  });

  test('主行新建标签页按钮存在', async ({ page }) => {
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 检查主行的+按钮
      const newTabBtn = card.locator('.fingerprint-actions .btn-icon.btn-new-tab');
      await expect(newTabBtn).toBeVisible();
    }
  });

  test('展开区域显示指纹详情', async ({ page }) => {
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 展开卡片
      await card.locator('.fingerprint-row').click();
      
      // 检查详情区域
      const details = card.locator('.fingerprint-details');
      await expect(details).toBeVisible();
      
      // 检查详情字段
      await expect(details.locator('.detail-label').first()).toBeVisible();
      await expect(details.locator('.detail-value').first()).toBeVisible();
    }
  });
});
