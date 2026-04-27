import playwright from "playwright";
import { randomUUID } from "crypto";
import fs from "fs";
import path from "path";

const SCREENSHOT_DIR = "./screenshots";

// 确保截图目录存在
if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

// Facebook 导航配置
const NAVIGATION_CONFIG = {
  baseUrl: "https://www.facebook.com",
  maxDepth: 5,
  sections: [
    { name: "home", url: "/" },
    { name: "messages", url: "/messages/" },
    { name: "friends", url: "/friends/" },
    { name: "groups", url: "/groups/" },
    { name: "pages", url: "/pages/" },
    { name: "events", url: "/events/" },
    { name: "marketplace", url: "/marketplace/" },
    { name: "saved", url: "/saved/" },
    { name: "notifications", url: "/notifications/" },
    { name: "settings", url: "/settings/" },
  ],
};

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function randomDelay(min = 1000, max = 3000) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function randomChoice(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

// ========== 人类行为模拟（使用注入的JS） ==========

/**
 * 随机滚动到页面某个区域 - 调用注入的 JS
 */
async function randomHumanScroll(page) {
  // 调用注入的人类行为模拟器
  await page.evaluate(() => {
    if (window.__humanBehavior) {
      return window.__humanBehavior.randomScroll();
    }
  });
}

/**
 * 人类行为滚动 - 调用注入的 JS
 */
async function humanScroll(page, scrollDistance = null) {
  if (scrollDistance === null) {
    const scrollHeight = await page.evaluate(() => document.body.scrollHeight - window.innerHeight);
    scrollDistance = Math.random() * scrollHeight * 0.5;
  }

  await page.evaluate((targetY) => {
    if (window.__humanBehavior) {
      return window.__humanBehavior.scrollTo(targetY);
    }
  }, scrollDistance);
}

/**
 * 人类行为点击元素 - 调用注入的 JS
 */
async function humanClick(page, selector) {
  const success = await page.evaluate((sel) => {
    if (window.__humanBehavior) {
      return window.__humanBehavior.clickElement(sel);
    }
    return false;
  }, selector);

  if (!success) {
    console.log(`   ⚠️ 元素 ${selector} 未找到或不可点击`);
  }
  return success;
}

/**
 * 人类思考/等待
 */
async function humanThink(page, minMs = 500, maxMs = 2000) {
  await page.evaluate(({ min, max }) => {
    if (window.__humanBehavior) {
      return window.__humanBehavior.think(min, max);
    }
  }, { min: minMs, max: maxMs });
}

async function takeScreenshot(page, name) {
  const timestamp = Date.now();
  const filename = `${name}-${timestamp}.png`;
  const filepath = path.join(SCREENSHOT_DIR, filename);
  await page.screenshot({ path: filepath, fullPage: false });
  console.log(`📸 截图: ${filename}`);
  return filepath;
}

async function getPageInfo(page) {
  try {
    return {
      url: page.url(),
      title: await page.title(),
      visibleText: await page
        .evaluate(() => document.body?.innerText?.substring(0, 500) || "")
        .catch(() => ""),
    };
  } catch {
    return { url: page.url(), title: "", visibleText: "" };
  }
}

async function extractNavigationLinks(page) {
  try {
    const links = await page.evaluate(() => {
      const results = [];
      // 查找导航链接
      const navSelectors = [
        'a[href*="/groups/"]',
        'a[href*="/pages/"]',
        'a[href*="/events/"]',
        'a[href*="/friends/"]',
        'a[href*="/marketplace/"]',
        'a[href*="/saved/"]',
        'a[href*="/settings/"]',
        'a[href*="/home/"]',
      ];

      const seen = new Set();
      navSelectors.forEach((selector) => {
        document.querySelectorAll(selector).forEach((a) => {
          const href = a.href;
          if (href && !seen.has(href) && href.includes("facebook.com")) {
            seen.add(href);
            results.push({
              href,
              text: a.innerText?.trim()?.substring(0, 50) || "",
            });
          }
        });
      });
      return results.slice(0, 20);
    });
    return links;
  } catch {
    return [];
  }
}

async function browseSection(page, section, browser) {
  console.log(`\n🌐 正在浏览: ${section.name} (${section.url})`);

  try {
    const fullUrl = `${NAVIGATION_CONFIG.baseUrl}${section.url}`;
    await page.goto(fullUrl, {
      timeout: 30000,
      waitUntil: "domcontentloaded",
    });

    // 等待页面加载
    await sleep(randomDelay(2000, 4000));

    // 🧑 人类行为：进入栏目后随机滚动
    await randomHumanScroll(page);
    await sleep(randomDelay(500, 1500));

    // 截图
    await takeScreenshot(page, `section-${section.name}`);

    // 获取页面信息
    const info = await getPageInfo(page);
    console.log(`   标题: ${info.title}`);
    console.log(`   链接: ${info.url}`);

    return { section, success: true, info };
  } catch (e) {
    console.log(`   ❌ 失败: ${e.message.substring(0, 100)}`);
    return { section, success: false, error: e.message };
  }
}

async function randomBrowse(page, depth = 0) {
  if (depth >= NAVIGATION_CONFIG.maxDepth) {
    console.log("✅ 达到最大深度限制，停止浏览");
    return;
  }

  console.log(`\n📍 当前深度: ${depth + 1}/${NAVIGATION_CONFIG.maxDepth}`);
  console.log(`   页面: ${page.url()}`);

  // 🧑 人类行为：随机滚动页面，模拟阅读
  await randomHumanScroll(page);
  await sleep(randomDelay(500, 1500));

  // 截图当前页面
  const pageName = `depth-${depth + 1}-${Date.now()}`;
  await takeScreenshot(page, pageName);

  // 提取可点击的链接
  const links = await extractNavigationLinks(page);
  console.log(`   发现 ${links.length} 个可访问链接`);

  if (links.length === 0) {
    console.log("   无更多链接可访问");
    return;
  }

  // 随机选择 1-3 个链接访问
  const numToVisit = Math.min(Math.floor(Math.random() * 3) + 1, links.length);
  const linksToVisit = [];

  while (linksToVisit.length < numToVisit) {
    const link = randomChoice(links);
    if (!linksToVisit.some((l) => l.href === link.href)) {
      linksToVisit.push(link);
    }
  }

  for (const link of linksToVisit) {
    console.log(`\n🔗 随机选择链接: ${link.text || link.href}`);

    try {
      // 🧑 人类行为：进入页面前回滚到顶部（调用注入的JS）
      if (Math.random() > 0.5) {
        await page.evaluate(() => {
          if (window.__humanBehavior) {
            return window.__humanBehavior.scrollTo(0);
          }
          window.scrollTo(0, 0);
        });
        await sleep(randomDelay(300, 800));
      }

      await page.goto(link.href, {
        timeout: 20000,
        waitUntil: "domcontentloaded",
      });

      await sleep(randomDelay(2000, 4000));

      // 🧑 人类行为：进入页面后随机滚动，模拟阅读
      await randomHumanScroll(page);

      // 截图
      const linkName = `link-${depth + 1}-${randomUUID().substring(0, 8)}`;
      await takeScreenshot(page, linkName);

      console.log(`   新页面: ${page.url()}`);

      // 递归深度浏览
      await randomBrowse(page, depth + 1);

      // 🧑 人类行为：返回前稍作停顿
      await sleep(randomDelay(500, 1500));

      // 返回原页面继续
      await page.goBack();
      await sleep(randomDelay(1000, 2000));
    } catch (e) {
      console.log(`   ❌ 访问失败: ${e.message.substring(0, 80)}`);
    }
  }
}

async function main() {
  console.log("🚀 启动 Facebook 浏览器...");
  console.log(`📁 截图保存目录: ${SCREENSHOT_DIR}`);

  const browser = await playwright.chromium.launch({
    headless: false, // 有头模式
    args: [
      "--start-maximized",
      "--disable-blink-features=AutomationControlled",
    ],
  });

  const context = await browser.newContext({
    viewport: { width: 1400, height: 900 },
    userAgent:
      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
  });

  const page = await context.newPage();

  // 反检测 + 人类行为模拟器（每次导航自动注入）
  await page.addInitScript(() => {
    // 1. 隐藏 webdriver
    Object.defineProperty(navigator, "webdriver", { get: () => false });

    // 2. 人类行为模拟器 - 会在每次页面导航后自动重新注入
    window.__humanBehavior = {
      // 鼠标移动
      async moveMouse(clientX, clientY, options = {}) {
        const steps = options.steps || (5 + Math.floor(Math.random() * 8));
        const startX = Math.random() * 100 + 50;
        const startY = Math.random() * 100 + 50;

        for (let i = 0; i <= steps; i++) {
          const t = i / steps;
          const x = startX + (clientX - startX) * t + (Math.random() - 0.5) * 2;
          const y = startY + (clientY - startY) * t + (Math.random() - 0.5) * 2;

          const evt = new MouseEvent('mousemove', {
            bubbles: true,
            cancelable: true,
            clientX: x,
            clientY: y
          });
          document.dispatchEvent(evt);
          await new Promise(r => setTimeout(r, 8 + Math.random() * 12));
        }
      },

      // 鼠标点击
      async click(clientX, clientY, options = {}) {
        // 悬停
        await this.moveMouse(clientX, clientY, { steps: 3 });
        await new Promise(r => setTimeout(r, 150 + Math.random() * 200));

        // mousedown
        document.dispatchEvent(new MouseEvent('mousedown', {
          bubbles: true, cancelable: true, clientX, clientY, button: 0
        }));
        await new Promise(r => setTimeout(r, 50 + Math.random() * 80));

        // mouseup
        document.dispatchEvent(new MouseEvent('mouseup', {
          bubbles: true, cancelable: true, clientX, clientY, button: 0
        }));
        await new Promise(r => setTimeout(r, 30 + Math.random() * 50));

        // click
        document.dispatchEvent(new MouseEvent('click', {
          bubbles: true, cancelable: true, clientX, clientY, button: 0
        }));
      },

      // 滚动页面
      async scrollTo(targetY, options = {}) {
        const chunks = options.chunks || 5;
        const currentY = window.scrollY;

        for (let i = 0; i < chunks; i++) {
          const progress = (i + 1) / chunks;
          const newY = currentY + (targetY - currentY) * progress * (0.8 + Math.random() * 0.4);
          window.scrollTo(0, newY);
          await new Promise(r => setTimeout(r, 80 + Math.random() * 120));
        }

        // 微调
        window.scrollTo(0, targetY + (Math.random() - 0.5) * 10);
      },

      // 随机滚动
      async randomScroll() {
        const maxScroll = document.body.scrollHeight - window.innerHeight;
        const targetY = Math.random() * maxScroll;
        await this.scrollTo(targetY);
        await new Promise(r => setTimeout(r, 500 + Math.random() * 1000));
      },

      // 点击元素
      async clickElement(selector) {
        const el = document.querySelector(selector);
        if (!el) return false;

        const box = el.getBoundingClientRect();
        const x = box.left + box.width / 2 + (Math.random() - 0.5) * box.width * 0.3;
        const y = box.top + box.height / 2 + (Math.random() - 0.5) * box.height * 0.3;

        await this.click(x, y);
        return true;
      },

      // 等待（人类思考）
      async think(minMs = 500, maxMs = 2000) {
        await new Promise(r => setTimeout(r, minMs + Math.random() * (maxMs - minMs)));
      }
    };

    console.log('[HumanBehavior] 人类行为模拟器已注入');
  });

  try {
    // 1. 打开 Facebook
    console.log("\n📱 打开 Facebook...");
    await page.goto("https://www.facebook.com", {
      timeout: 30000,
      waitUntil: "domcontentloaded",
    });
    await sleep(3000);

    // 2. 检查是否需要登录
    const url = page.url();
    if (url.includes("login") || url.includes("checkpoint")) {
      console.log("\n⚠️ 需要登录!");
      console.log("请在浏览器中扫码或输入账号密码登录...");

      // 等待用户登录
      await page.waitForURL("**/facebook.com/**", { timeout: 0 }).catch(() => {});

      // 等待一段时间让用户登录
      console.log("等待登录中（请在浏览器中完成登录）...");
      await sleep(30000); // 等待30秒

      // 检查是否登录成功
      if (page.url().includes("login")) {
        console.log("❌ 登录超时，请重新运行脚本");
        await browser.close();
        return;
      }
    }

    console.log("\n✅ 登录成功!");
    console.log(`   当前页面: ${page.url()}`);

    // 截图主页
    await takeScreenshot(page, "00-homepage");

    // 3. 浏览各主要栏目
    console.log("\n📋 开始浏览主要栏目...");

    for (const section of NAVIGATION_CONFIG.sections.slice(0, 6)) {
      await browseSection(page, section, browser);
      await sleep(randomDelay(2000, 4000));
    }

    // 4. 随机深度浏览
    console.log("\n🎲 开始随机深度浏览...");
    await randomBrowse(page, 0);

    console.log("\n🎉 浏览完成!");
    console.log(`📁 截图保存在: ${SCREENSHOT_DIR}`);
  } catch (e) {
    console.error("❌ 发生错误:", e.message);
  } finally {
    console.log("\n按 Enter 键关闭浏览器...");
    await sleep(3000);
    await browser.close();
  }
}

main();
