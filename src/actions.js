function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

export async function humanPause(page, min = 300, max = 1500) {
  await page.waitForTimeout(randomInt(min, max));
}

export async function maybeMouseMove(page) {
  const width = page.viewportSize()?.width || 1280;
  const height = page.viewportSize()?.height || 800;
  await page.mouse.move(randomInt(20, width - 20), randomInt(20, height - 20), {
    steps: randomInt(5, 20),
  });
}

export async function actionLogin(page, config, account) {
  await page.goto(`${config.baseUrl}${config.loginPath}`, { waitUntil: "domcontentloaded" });
  await humanPause(page, 400, 1200);

  const username = page.locator(config.selectors.username).first();
  const password = page.locator(config.selectors.password).first();
  const submit = page.locator(config.selectors.loginSubmit).first();

  const hasUsername = (await username.count()) > 0;
  const hasPassword = (await password.count()) > 0;
  const hasSubmit = (await submit.count()) > 0;

  if (!hasUsername || !hasPassword || !hasSubmit) {
    throw new Error("login-selectors-not-found");
  }

  await username.fill(account.username);
  await humanPause(page, 200, 800);
  await password.fill(account.password);
  await humanPause(page, 300, 1000);
  await submit.click();
  await page.waitForLoadState("networkidle", { timeout: 15000 }).catch(() => {});
}

export async function actionBrowseFeed(page, config) {
  await page.goto(config.baseUrl, { waitUntil: "domcontentloaded" });
  await humanPause(page, 600, 1800);
  await maybeMouseMove(page);
  await page.mouse.wheel(0, randomInt(200, 1200));
  await humanPause(page, 500, 1600);
}

export async function actionSearch(page, config, query = "tech") {
  const input = page.locator(config.selectors.searchInput).first();
  if ((await input.count()) === 0) return;
  await input.click();
  await humanPause(page, 200, 700);
  await input.fill(query);
  await humanPause(page, 200, 700);
  await page.keyboard.press("Enter");
  await page.waitForLoadState("domcontentloaded").catch(() => {});
  await humanPause(page, 500, 1500);
}

export async function actionCreatePost(page, config, text) {
  const composer = page.locator(config.selectors.composer).first();
  if ((await composer.count()) === 0) return;
  await composer.click();
  await humanPause(page, 200, 900);
  await composer.fill(text);
  await humanPause(page, 200, 900);

  const btn = page.locator(config.selectors.postSubmit).first();
  if ((await btn.count()) > 0) {
    await btn.click();
    await page.waitForLoadState("networkidle").catch(() => {});
  }
}

export async function actionLikeTopPost(page, config) {
  const btn = page.locator(config.selectors.likeBtn).first();
  if ((await btn.count()) === 0) return;
  await btn.scrollIntoViewIfNeeded().catch(() => {});
  await humanPause(page, 200, 700);
  await btn.click();
  await humanPause(page, 300, 900);
}

export async function actionCommentTopPost(page, config, text) {
  const input = page.locator(config.selectors.commentInput).first();
  if ((await input.count()) === 0) return;
  await input.click();
  await humanPause(page, 150, 500);
  await input.fill(text);
  await humanPause(page, 150, 500);

  const submit = page.locator(config.selectors.commentSubmit).first();
  if ((await submit.count()) > 0) {
    await submit.click();
  } else {
    await page.keyboard.press("Enter");
  }
  await humanPause(page, 300, 900);
}

export async function actionOpenProfile(page, config) {
  const link = page.locator(config.selectors.profileLink).first();
  if ((await link.count()) === 0) return;
  await link.click();
  await page.waitForLoadState("domcontentloaded").catch(() => {});
  await humanPause(page, 500, 1200);
}

export const builtInActions = {
  login: actionLogin,
  browse_feed: actionBrowseFeed,
  search: actionSearch,
  create_post: actionCreatePost,
  like_top_post: actionLikeTopPost,
  comment_top_post: actionCommentTopPost,
  open_profile: actionOpenProfile,
};
