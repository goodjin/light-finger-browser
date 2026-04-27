import playwright from "playwright";
import fs from "fs";
import path from "path";
import { exec } from "child_process";

// 按日期创建子目录
const today = new Date().toISOString().split('T')[0];
const SCREENSHOT_DIR = `./screenshots/${today}`;
const SESSION_FILE = "./facebook-session.json";

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function randomDelay(min = 1000, max = 3000) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// ========== 正常人的浏览策略 ==========

const PRIORITY_ORDER = [
  'notification',  // 通知
  'message',      // 消息
  'friend',       // 好友
  'group',        // 小组
  'event',        // 活动
  'video',        // 视频
  'marketplace',  // 市场
];

// ========== 注入的 JS ==========

const INJECTED_SCRIPT = `
window.__fbAgent = {
  visitedUrls: new Set(),

  analyzePage() {
    const results = {
      sections: [],
      contentLinks: [],
      pageInfo: { title: document.title, url: window.location.href }
    };

    const seenUrls = new Set();

    document.querySelectorAll('a[href]').forEach(a => {
      const href = a.href;
      const text = a.innerText?.trim() || '';
      const ariaLabel = a.getAttribute('aria-label') || '';

      if (!href || !href.includes('facebook.com') || seenUrls.has(href)) return;
      if (href.includes('l.facebook.com')) return;
      if (text.length === 0 && ariaLabel.length === 0) return;

      seenUrls.add(href);
      const fullText = (text + ' ' + ariaLabel).trim();

      results.sections.push({
        text: fullText.substring(0, 80),
        href,
        type: this.categorize(href, fullText),
        isVisible: this.isVisible(a)
      });
    });

    // 去重
    const seenTexts = new Set();
    results.sections = results.sections.filter(s => {
      const key = s.text.toLowerCase().substring(0, 15);
      if (seenTexts.has(key)) return false;
      seenTexts.add(key);
      return true;
    });

    return results;
  },

  isVisible(el) {
    const box = el.getBoundingClientRect();
    return box && box.width > 0 && box.height > 0 &&
           box.top < window.innerHeight && box.bottom > 0;
  },

  categorize(href, text) {
    const url = href.toLowerCase();
    const txt = text.toLowerCase();

    if (url.includes('notifications')) return 'notification';
    if (url.includes('messages')) return 'message';
    if (url.includes('friends')) return 'friend';
    if (url.includes('groups')) return 'group';
    if (url.includes('watch')) return 'video';
    if (url.includes('marketplace')) return 'marketplace';
    if (url.includes('events')) return 'event';
    if (url.includes('pages')) return 'page';
    if (url.includes('photo')) return 'photo';
    if (url.includes('profile')) return 'profile';
    if (url.includes('settings')) return 'settings';

    if (txt.includes('通知')) return 'notification';
    if (txt.includes('消息') || txt.includes('messenger')) return 'message';
    if (txt.includes('朋友') || txt.includes('好友')) return 'friend';
    if (txt.includes('小组') || txt.includes('社群')) return 'group';
    if (txt.includes('视频')) return 'video';
    if (txt.includes('市场')) return 'marketplace';
    if (txt.includes('活动')) return 'event';

    return 'other';
  },

  async humanClick(x, y) {
    for (let i = 0; i < 6; i++) {
      const t = i / 5;
      const cx = (Math.random() * 200 + 50) + (x - (Math.random() * 200 + 50)) * t;
      const cy = (Math.random() * 200 + 50) + (y - (Math.random() * 200 + 50)) * t;
      document.dispatchEvent(new MouseEvent('mousemove', {
        bubbles: true, clientX: cx, clientY: cy
      }));
      await new Promise(r => setTimeout(r, 10 + Math.random() * 10));
    }

    await new Promise(r => setTimeout(r, 150 + Math.random() * 200));

    ['mousedown', 'mouseup', 'click'].forEach(type => {
      document.dispatchEvent(new MouseEvent(type, {
        bubbles: true, clientX: x, clientY: y, button: 0
      }));
    });
  },

  async humanScroll() {
    const maxScroll = document.body.scrollHeight - window.innerHeight;
    const target = Math.random() * maxScroll;
    const current = window.scrollY;
    const chunks = 5;

    for (let i = 0; i < chunks; i++) {
      const t = (i + 1) / chunks;
      window.scrollTo(0, current + (target - current) * t);
      await new Promise(r => setTimeout(r, 100 + Math.random() * 100));
    }
  },

  async goTo(href) {
    const el = document.querySelector(\`a[href="\${href}"]\`) ||
               Array.from(document.querySelectorAll('a')).find(a => a.href === href);
    if (!el) return false;

    const box = el.getBoundingClientRect();
    if (!box || box.width === 0) return false;

    const x = box.left + box.width / 2 + (Math.random() - 0.5) * box.width * 0.4;
    const y = box.top + box.height / 2 + (Math.random() - 0.5) * box.height * 0.4;

    this.visitedUrls.add(href);
    await this.humanClick(x, y);
    return true;
  },

  // 检测是否需要登录
  checkLoginRequired() {
    const url = window.location.href;
    const title = document.title.toLowerCase();
    const bodyText = document.body.innerText.toLowerCase();

    if (url.includes('login') || url.includes('checkpoint')) return true;
    if (title.includes('登录') || title.includes('login')) return true;
    if (bodyText.includes('登录facebook') || bodyText.includes('log in to facebook')) return true;

    return false;
  }
};
`;

// ========== 主程序 ==========

async function takeScreenshot(page, name) {
  const timestamp = Date.now();
  const filename = `${name}-${timestamp}.png`;
  const filepath = path.join(SCREENSHOT_DIR, filename);
  await page.screenshot({ path: filepath, fullPage: false });
  console.log(`📸 截图: ${filename}`);
  return filepath;
}

function saveSession(context) {
  try {
    // 保存 context 的 cookies 和 storage state
    const storageState = context.storageState();
    // 只保存 cookies（不包含 localStorage 等）
    fs.writeFileSync(SESSION_FILE, JSON.stringify({
      cookies: context.cookies(),
      timestamp: Date.now()
    }, null, 2));
    console.log("💾 会话已保存");
  } catch (e) {
    console.log("⚠️ 保存会话失败:", e.message);
  }
}

async function loadSession(context) {
  try {
    if (fs.existsSync(SESSION_FILE)) {
      const data = JSON.parse(fs.readFileSync(SESSION_FILE, 'utf8'));
      if (data.cookies && data.cookies.length > 0) {
        await context.addCookies(data.cookies);
        console.log("✅ 已加载保存的会话");
        return true;
      }
    }
  } catch (e) {
    console.log("⚠️ 加载会话失败:", e.message);
  }
  return false;
}

async function checkChromeRunning() {
  return new Promise((resolve) => {
    exec("lsof -i :9922 2>/dev/null | grep -q Google", (err) => {
      resolve(!err);
    });
  });
}

async function main() {
  console.log("🚀 Facebook 智能代理启动\n");

  let browser, context, page;
  const visitedUrls = new Set();
  const needLogin = { value: false };

  try {
    // 检查是否有 Chrome 在调试模式下运行
    const chromeRunning = await checkChromeRunning();

    if (chromeRunning) {
      console.log("📡 连接到已运行的 Chrome (调试模式)...");

      try {
        browser = await playwright.chromium.connectOverCDP('http://127.0.0.1:9922');
        context = browser.contexts()[0];
        page = context.pages().find(p => !p.url().includes('chrome://') && !p.url().includes('chrome-extension://'));

        if (!page) {
          // 创建一个新标签页
          page = await context.newPage();
        }

        // 在新标签页打开 Facebook
        await page.goto("https://www.facebook.com", {
          timeout: 20000,
          waitUntil: "domcontentloaded"
        });
      } catch (e) {
        console.log("⚠️ 连接失败，尝试启动新浏览器...");
        browser = null;
      }
    }

    // 如果没有连接的浏览器，启动新浏览器
    if (!browser) {
      // 尝试使用用户默认的 Chrome 配置
      const userDataDir = process.env.HOME + '/Library/Application Support/Google/Chrome';

      console.log("🔍 查找 Chrome 配置...");

      // 检查是否有 Default 配置
      if (fs.existsSync(userDataDir)) {
        console.log("✅ 找到用户 Chrome 配置");

        // 由于 Chrome 不能同时用同一配置启动多个实例，我们用临时配置
        const tempProfileDir = '/tmp/facebook-agent-' + process.env.USER;
        if (!fs.existsSync(tempProfileDir)) {
          fs.mkdirSync(tempProfileDir, { recursive: true });
        }

        console.log("🔧 启动 Chrome (使用临时配置目录)...");

        browser = await playwright.chromium.launch({
          headless: false,
          args: [
            `--user-data-dir=${tempProfileDir}`,
            '--new-tab',
            '--start-maximized'
          ],
          executablePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'
        });
      } else {
        // 没有找到用户配置，启动普通浏览器
        console.log("📦 启动独立 Chrome...");
        browser = await playwright.chromium.launch({
          headless: false,
          args: ["--start-maximized"]
        });
      }

      context = await browser.newContext({
        viewport: { width: 1400, height: 900 },
      });

      // 尝试加载保存的会话
      await loadSession(context);

      page = await context.newPage();
    }

    // 注入基础脚本（CDP连接时用evaluate）
    await page.evaluate(() => {
      Object.defineProperty(navigator, "webdriver", { get: () => false });
    });

    console.log("📱 打开 Facebook...");

    // 检测是否需要登录
    const checkAndHandleLogin = async () => {
      const url = page.url();
      const title = await page.title();

      if (url.includes('login') || title.toLowerCase().includes('login') ||
          title.includes('登录') || url.includes('checkpoint')) {
        console.log("\n" + "=".repeat(50));
        console.log("⚠️  需要登录!");
        console.log("=".repeat(50));
        console.log("请在浏览器中完成登录...");
        console.log("登录后按 Enter 继续...");
        console.log("=".repeat(50));

        needLogin.value = true;

        // 等待用户按 Enter
        await new Promise(resolve => {
          const rl = require('readline').createInterface({
            input: process.stdin,
            output: process.stdout
          });
          rl.question('', () => {
            rl.close();
            resolve();
          });
        });

        // 保存会话
        saveSession(context);
        needLogin.value = false;
        return true;
      }
      return false;
    };

    // 打开 Facebook
    try {
      await page.goto("https://www.facebook.com", {
        timeout: 20000,
        waitUntil: "domcontentloaded"
      });
      await sleep(2000);

      if (await checkAndHandleLogin()) {
        // 登录后等待页面加载
        await sleep(3000);
      }
    } catch (e) {
      console.log("⚠️ 页面加载超时，等待用户操作...");
      await sleep(5000);
      await checkAndHandleLogin();
    }

    // 检查当前 URL
    const currentUrl = page.url();
    console.log(`\n✅ 当前页面: ${currentUrl}\n`);

    if (currentUrl.includes('login')) {
      await checkAndHandleLogin();
    }

    // 📸 截图：登录后的首页
    await takeScreenshot(page, "00-homepage");

    // 注入分析脚本（CDP连接时用evaluate代替addInitScript）
    console.log("🔍 注入分析脚本...");
    await page.evaluate((script) => {
      eval(script);
    }, INJECTED_SCRIPT);
    await sleep(1000);

    // 分析页面
    console.log("📊 分析页面结构...\n");

    const analysis = await page.evaluate(() => {
      if (window.__fbAgent) {
        return window.__fbAgent.analyzePage();
      }
      return null;
    });

    if (!analysis) {
      console.log("❌ 分析脚本加载失败");
      console.log("\n浏览器保持打开状态，你可以手动浏览。");
      return;
    }

    // 过滤可见栏目
    const visibleSections = analysis.sections.filter(s => s.isVisible);
    console.log(`发现 ${visibleSections.length} 个可见栏目\n`);

    // 📸 截图：分析后的页面
    await takeScreenshot(page, "00-analysis");

    // 按类型分组显示
    const byType = {};
    visibleSections.forEach(s => {
      if (!byType[s.type]) byType[s.type] = [];
      byType[s.type].push(s);
    });

    console.log("=" .repeat(50));
    console.log("📋 发现的栏目:");
    console.log("=".repeat(50));

    Object.keys(byType).forEach(type => {
      console.log(`\n【${type.toUpperCase()}】 (${byType[type].length}个)`);
      byType[type].slice(0, 3).forEach(s => {
        console.log(`  • ${s.text.substring(0, 40)}`);
      });
    });

    // 代理决策
    console.log("\n" + "=".repeat(50));
    console.log("🤔 代理决策...\n");

    let selected = null;
    for (const type of PRIORITY_ORDER) {
      const candidates = visibleSections.filter(s =>
        s.type === type && !visitedUrls.has(s.href)
      );
      if (candidates.length > 0) {
        selected = candidates[Math.floor(Math.random() * candidates.length)];
        console.log(`✅ 选择: ${selected.text}`);
        console.log(`   类型: ${selected.type}`);
        break;
      }
    }

    if (!selected) {
      const unvisited = visibleSections.filter(s => !visitedUrls.has(s.href));
      if (unvisited.length > 0) {
        selected = unvisited[Math.floor(Math.random() * unvisited.length)];
        console.log(`✅ 随机选择: ${selected.text}`);
      }
    }

    if (selected) {
      console.log("\n🌐 访问中...");
      visitedUrls.add(selected.href);

      const success = await page.evaluate((href) => {
        if (window.__fbAgent) {
          return window.__fbAgent.goTo(href);
        }
        return false;
      }, selected.href);

      if (success) {
        await sleep(3000);
        console.log(`📍 已跳转到: ${page.url()}`);
        await takeScreenshot(page, `01-selected-${selected.type}`);
        await takeScreenshot(page, `02-visited-${Date.now()}`);

        // 检测是否需要登录
        await checkAndHandleLogin();
      }
    }

    console.log("\n" + "=".repeat(50));
    console.log("✅ 首次访问完成!");
    console.log("=".repeat(50));
    console.log("\n浏览器将保持打开状态，你可以继续使用。");
    console.log("随时可以运行脚本继续代理操作。");

    // 不关闭浏览器，保持用户使用
    console.log("\n按 Ctrl+C 退出脚本（浏览器会保持打开）...");

    // 等待用户
    await new Promise(() => {});

  } catch (e) {
    console.error("❌ 发生错误:", e.message);
    console.log("\n浏览器保持打开状态。");
  }
}

main();
