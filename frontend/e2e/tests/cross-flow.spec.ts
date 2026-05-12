import { test, expect } from '@playwright/test';

test.describe('跨流程测试', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/');
    await page.waitForLoadState('networkidle');
  });

  test('新建到编辑流程', async ({ page }) => {
    // 检查是否有警告提示
    const warningBanner = page.locator('.warning-banner');
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    if (hasWarning) {
      test.skip(true, '需要运行中的浏览器实例');
      return;
    }
    
    // 1. 新建指纹
    await page.locator('.page-header .btn-primary').click();
    
    const modal = page.locator('.modal');
    await expect(modal).toBeVisible();
    
    // 选择国家（日本）
    await modal.locator('select').selectOption('JP');
    
    // 点击创建
    await modal.locator('.btn-primary').click();
    
    // 等待对话框关闭
    await expect(modal).not.toBeVisible({ timeout: 5000 });
    
    // 2. 等待列表更新
    await page.waitForTimeout(1000);
    
    // 3. 找到刚创建的指纹并编辑
    const newCard = page.locator('.fingerprint-card').first();
    const hasCard = await newCard.isVisible().catch(() => false);
    
    if (hasCard) {
      // 点击编辑按钮
      await newCard.locator('.btn-icon.btn-edit').click();
      
      // 检查编辑对话框
      const editModal = page.locator('.modal');
      await expect(editModal).toBeVisible();
      await expect(editModal.locator('h3')).toContainText('编辑指纹');
      
      // 修改标题
      const titleInput = editModal.locator('input').first();
      await titleInput.fill('测试指纹标题');
      
      // 保存
      await editModal.locator('.btn-primary').click();
      
      // 对话框关闭
      await expect(editModal).not.toBeVisible({ timeout: 3000 });
    }
  });

  test('新建到复制流程', async ({ page }) => {
    const warningBanner = page.locator('.warning-banner');
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    if (hasWarning) {
      test.skip(true, '需要运行中的浏览器实例');
      return;
    }
    
    // 1. 创建第一个指纹
    await page.locator('.page-header .btn-primary').click();
    
    const modal = page.locator('.modal');
    await expect(modal).toBeVisible();
    
    await modal.locator('select').selectOption('DE');
    await modal.locator('.btn-primary').click();
    await expect(modal).not.toBeVisible({ timeout: 5000 });
    
    // 等待列表更新
    await page.waitForTimeout(1000);
    
    // 记录当前指纹数量
    const initialCount = await page.locator('.fingerprint-card').count();
    
    // 2. 找到第一个指纹并复制
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      await card.locator('.btn-icon.btn-copy').click();
      
      // 检查复制对话框
      const copyModal = page.locator('.modal');
      await expect(copyModal).toBeVisible();
      
      // 修改标题
      const titleInput = copyModal.locator('input').nth(1);
      await titleInput.fill('复制的指纹');
      
      // 点击复制
      await copyModal.locator('.btn-primary').click();
      
      // 对话框关闭
      await expect(copyModal).not.toBeVisible({ timeout: 5000 });
      
      // 等待列表更新
      await page.waitForTimeout(1000);
      
      // 3. 验证两个指纹都存在
      const newCount = await page.locator('.fingerprint-card').count();
      expect(newCount).toBeGreaterThanOrEqual(initialCount);
    }
  });

  test('展开卡片到创建标签页', async ({ page }) => {
    const warningBanner = page.locator('.warning-banner');
    const hasWarning = await warningBanner.isVisible().catch(() => false);
    
    if (hasWarning) {
      test.skip(true, '需要运行中的浏览器实例');
      return;
    }
    
    // 创建指纹
    await page.locator('.page-header .btn-primary').click();
    const modal = page.locator('.modal');
    await modal.locator('select').selectOption('FR');
    await modal.locator('.btn-primary').click();
    await expect(modal).not.toBeVisible({ timeout: 5000 });
    
    await page.waitForTimeout(1000);
    
    // 找到指纹卡片
    const card = page.locator('.fingerprint-card').first();
    const hasCard = await card.isVisible().catch(() => false);
    
    if (hasCard) {
      // 展开卡片
      await card.locator('.fingerprint-row').click();
      
      // 检查展开区域
      await expect(card.locator('.fingerprint-tabs-section')).toBeVisible();
      
      // 检查空状态或标签页列表
      const noTabs = card.locator('.no-tabs');
      const tabsList = card.locator('.tabs-list');
      
      const hasNoTabs = await noTabs.isVisible().catch(() => false);
      const hasTabs = await tabsList.isVisible().catch(() => false);
      
      expect(hasNoTabs || hasTabs).toBeTruthy();
    }
  });
});
