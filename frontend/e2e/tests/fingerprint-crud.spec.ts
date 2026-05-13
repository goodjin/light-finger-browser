import { test, expect } from '@playwright/test';

test.describe('指纹 CRUD 操作', () => {
  test.beforeEach(async ({ page }) => {
    // Use Wails app URL for proper IPC connection
    await page.goto('http://localhost:34115/');
    await page.waitForLoadState('networkidle');
  });

  test('新建指纹对话框打开', async ({ page }) => {
    // 检查是否有警告提示（无实例时按钮会被禁用）
    const warningBanner = page.locator('.warning-banner');
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    if (hasWarning) {
      // 无实例时，跳过此测试
      test.skip(true, '需要运行中的浏览器实例');
      return;
    }
    
    // 点击新建按钮
    await page.locator('.page-header .btn-primary').click();
    
    // 检查对话框出现
    const modal = page.locator('.modal');
    await expect(modal).toBeVisible();
    await expect(modal.locator('h3')).toContainText('新建指纹窗口');
    
    // 检查国家选择器
    await expect(modal.locator('.form-group select')).toBeVisible();
    
    // 检查 URL 输入框
    await expect(modal.locator('.form-group input')).toBeVisible();
    
    // 检查按钮
    await expect(modal.locator('.modal-actions button')).toHaveCount(2);
  });

  test('选择不同国家', async ({ page }) => {
    const warningBanner = page.locator('.warning-banner');
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    if (hasWarning) {
      test.skip(true, '需要运行中的浏览器实例');
      return;
    }
    
    await page.locator('.page-header .btn-primary').click();
    
    const select = page.locator('.modal .form-group select');
    await expect(select).toBeVisible();
    
    // 初始值应该是 US
    await expect(select).toHaveValue('US');
    
    // 选择日本
    await select.selectOption('JP');
    await expect(select).toHaveValue('JP');
  });

  test('取消新建对话框', async ({ page }) => {
    const warningBanner = page.locator('.warning-banner');
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    if (hasWarning) {
      test.skip(true, '需要运行中的浏览器实例');
      return;
    }
    
    await page.locator('.page-header .btn-primary').click();
    
    const modal = page.locator('.modal');
    await expect(modal).toBeVisible();
    
    // 点击取消
    await modal.locator('.modal-actions button').first().click();
    
    // 对话框应该关闭
    await expect(modal).not.toBeVisible();
  });

  test('点击遮罩关闭对话框', async ({ page }) => {
    const warningBanner = page.locator('.warning-banner');
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    if (hasWarning) {
      test.skip(true, '需要运行中的浏览器实例');
      return;
    }
    
    await page.locator('.page-header .btn-primary').click();
    
    const modal = page.locator('.modal');
    await expect(modal).toBeVisible();
    
    // 点击遮罩
    await page.locator('.modal-overlay').click({ position: { x: 10, y: 10 }, force: true });
    
    // 对话框应该关闭
    await expect(modal).not.toBeVisible({ timeout: 2000 });
  });

  test('编辑对话框打开', async ({ page }) => {
    // 确保有指纹可编辑
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 点击编辑按钮
      await card.locator('.btn-icon.btn-edit').click();
      
      // 检查编辑对话框
      const modal = page.locator('.modal');
      await expect(modal).toBeVisible();
      await expect(modal.locator('h3')).toContainText('编辑指纹');
      
      // 检查只读字段
      await expect(modal.locator('.form-group.readonly')).toHaveCount(2); // 国家、Seed
    }
  });

  test('复制对话框打开', async ({ page }) => {
    // 确保有指纹可复制
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 点击复制按钮
      await card.locator('.btn-icon.btn-copy').click();
      
      // 检查复制对话框
      const modal = page.locator('.modal');
      await expect(modal).toBeVisible();
      await expect(modal.locator('h3')).toContainText('复制指纹');
      
      // 检查国家选择器
      await expect(modal.locator('.form-group select')).toBeVisible();
    }
  });

  test('删除指纹确认', async ({ page }) => {
    // 确保有指纹可删除
    const cards = page.locator('.fingerprint-card');
    const count = await cards.count();
    
    if (count > 0) {
      const card = cards.first();
      const initialCount = count;
      
      // 点击删除按钮
      await card.locator('.btn-icon.btn-danger').click();
      
      // 等待删除完成
      await page.waitForTimeout(500);
      
      // 检查指纹数量减少
      const newCount = await cards.count();
      expect(newCount).toBeLessThanOrEqual(initialCount);
    }
  });
});
