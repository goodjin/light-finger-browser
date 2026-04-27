#!/usr/bin/env node
import { chromium } from "playwright";
import { config } from "./config.js";
import { builtInActions } from "./actions.js";
import { chooseNextAction } from "./llmPolicy.js";
import { createRunLogger } from "./logger.js";

function parseArgs(argv) {
  return {
    headed: argv.includes("--headed"),
  };
}

function pickText(type, payload, account, step) {
  if (type === "post") {
    return (
      payload?.postText ||
      `Testing post ${step + 1} by ${account.username} at ${new Date().toISOString()}`
    );
  }
  if (type === "comment") {
    return payload?.commentText || `Interesting point! test-step-${step + 1}`;
  }
  if (type === "search") {
    return payload?.searchQuery || ["product", "news", "friends", "video"][step % 4];
  }
  return "";
}

async function runUserSession({ browser, account, maxMs, logger, runConfig }) {
  const context = await browser.newContext({
    viewport: {
      width: 1200 + Math.floor(Math.random() * 200),
      height: 760 + Math.floor(Math.random() * 200),
    },
  });
  const page = await context.newPage();

  const sessionStart = Date.now();
  let step = 0;
  const recentActions = [];

  logger.logEvent({ type: "session_start", userId: account.id, username: account.username });

  try {
    const loginStart = Date.now();
    try {
      await builtInActions.login(page, runConfig, account);
      logger.logEvent({
        type: "action",
        userId: account.id,
        action: "login",
        ok: true,
        durationMs: Date.now() - loginStart,
      });
    } catch (err) {
      logger.logEvent({
        type: "action",
        userId: account.id,
        action: "login",
        ok: false,
        error: err?.message || String(err),
        durationMs: Date.now() - loginStart,
      });
    }

    while (Date.now() - sessionStart < maxMs) {
      const decision = await chooseNextAction({
        config: runConfig,
        account,
        stepIndex: step,
        recentActions: recentActions.slice(-5),
      });

      const actionName = decision.action;
      const fn = builtInActions[actionName];
      if (!fn) {
        logger.logEvent({
          type: "action",
          userId: account.id,
          action: actionName,
          ok: false,
          error: "unknown-action",
          durationMs: 0,
        });
        step += 1;
        continue;
      }

      const started = Date.now();
      try {
        if (actionName === "create_post") {
          await fn(page, runConfig, pickText("post", decision.payload, account, step));
        } else if (actionName === "comment_top_post") {
          await fn(page, runConfig, pickText("comment", decision.payload, account, step));
        } else if (actionName === "search") {
          await fn(page, runConfig, pickText("search", decision.payload, account, step));
        } else {
          await fn(page, runConfig);
        }

        logger.logEvent({
          type: "action",
          userId: account.id,
          action: actionName,
          ok: true,
          reason: decision.reason,
          durationMs: Date.now() - started,
        });
      } catch (err) {
        logger.logEvent({
          type: "action",
          userId: account.id,
          action: actionName,
          ok: false,
          reason: decision.reason,
          error: err?.message || String(err),
          durationMs: Date.now() - started,
        });
      }

      recentActions.push(actionName);
      step += 1;
    }
  } finally {
    logger.logEvent({ type: "session_end", userId: account.id, username: account.username });
    await context.close();
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const logger = createRunLogger();

  if (!config.accounts.length) {
    console.error("No accounts found. Please configure TEST_ACCOUNTS in .env");
    process.exit(1);
  }

  const browser = await chromium.launch({ headless: args.headed ? false : config.headless });

  const durationMs = config.runMinutes * 60 * 1000;
  const accounts = config.accounts.slice(0, Math.max(1, config.concurrentUsers));

  try {
    await Promise.all(
      accounts.map((account) =>
        runUserSession({
          browser,
          account,
          maxMs: durationMs,
          logger,
          runConfig: config,
        })
      )
    );
  } finally {
    await browser.close();
  }

  const result = logger.finalize();
  const summary = result.state.summary;
  console.log("Run completed");
  console.log(`Report: ${result.file}`);
  console.log(
    `Users=${summary.users} Actions=${summary.actions} Success=${summary.success} Failed=${summary.failed} AvgStepMs=${summary.avgStepMs}`
  );
}

main().catch((err) => {
  console.error("Runner failed", err);
  process.exit(1);
});
