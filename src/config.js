import dotenv from "dotenv";

dotenv.config();

function toBool(value, fallback) {
  if (value === undefined || value === null || value === "") return fallback;
  return String(value).toLowerCase() === "true";
}

function toInt(value, fallback) {
  const n = Number.parseInt(value, 10);
  return Number.isFinite(n) ? n : fallback;
}

function parseAccounts(raw) {
  if (!raw) return [];
  return raw
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean)
    .map((pair, idx) => {
      const [username, password] = pair.split(":");
      return {
        id: `acct-${idx + 1}`,
        username: username?.trim() || `user${idx + 1}`,
        password: password?.trim() || "",
      };
    });
}

export const config = {
  baseUrl: process.env.BASE_URL || "http://localhost:3000",
  loginPath: process.env.LOGIN_PATH || "/login",
  headless: toBool(process.env.HEADLESS, true),
  runMinutes: toInt(process.env.RUN_MINUTES, 5),
  concurrentUsers: toInt(process.env.CONCURRENT_USERS, 3),
  llm: {
    apiKey: process.env.OPENAI_API_KEY || "",
    baseUrl: process.env.OPENAI_BASE_URL || "https://api.openai.com/v1",
    model: process.env.OPENAI_MODEL || "gpt-4.1-mini",
  },
  accounts: parseAccounts(process.env.TEST_ACCOUNTS),
  selectors: {
    username: '[name="username"], [data-testid="login-username"]',
    password: '[name="password"], [data-testid="login-password"]',
    loginSubmit:
      'button[type="submit"], [data-testid="login-submit"], button:has-text("Login")',
    composer:
      '[data-testid="composer-input"], textarea[name="post"], textarea[placeholder*="What" i]',
    postSubmit:
      '[data-testid="composer-submit"], button:has-text("Post"), button:has-text("发布")',
    feedItems: '[data-testid="feed-item"], article, .post-card',
    likeBtn:
      '[data-testid="like-btn"], button:has-text("Like"), button:has-text("赞")',
    commentInput:
      '[data-testid="comment-input"], textarea[name="comment"], input[name="comment"]',
    commentSubmit:
      '[data-testid="comment-submit"], button:has-text("Comment"), button:has-text("评论")',
    searchInput: '[data-testid="search-input"], input[type="search"], input[name="q"]',
    profileLink:
      '[data-testid="nav-profile"], a[href*="/profile"], a:has-text("Profile")',
  },
};
