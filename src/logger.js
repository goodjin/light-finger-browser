import fs from "fs";
import path from "path";

const outDir = path.resolve(process.cwd(), "output");

export function ensureOutputDir() {
  if (!fs.existsSync(outDir)) fs.mkdirSync(outDir, { recursive: true });
  return outDir;
}

export function createRunLogger() {
  const dir = ensureOutputDir();
  const startedAt = new Date();
  const runId = startedAt.toISOString().replace(/[:.]/g, "-");
  const file = path.join(dir, `run-${runId}.json`);

  const state = {
    runId,
    startedAt: startedAt.toISOString(),
    endedAt: null,
    events: [],
    summary: {
      users: 0,
      actions: 0,
      success: 0,
      failed: 0,
      avgStepMs: 0,
    },
  };

  function logEvent(event) {
    state.events.push({
      at: new Date().toISOString(),
      ...event,
    });
  }

  function finalize() {
    state.endedAt = new Date().toISOString();
    const actionEvents = state.events.filter((e) => e.type === "action");
    state.summary.actions = actionEvents.length;
    state.summary.success = actionEvents.filter((e) => e.ok).length;
    state.summary.failed = actionEvents.filter((e) => !e.ok).length;
    state.summary.users = new Set(state.events.map((e) => e.userId).filter(Boolean)).size;

    const totalStepMs = actionEvents.reduce((sum, e) => sum + (e.durationMs || 0), 0);
    state.summary.avgStepMs = actionEvents.length
      ? Number((totalStepMs / actionEvents.length).toFixed(2))
      : 0;

    fs.writeFileSync(file, JSON.stringify(state, null, 2), "utf-8");
    return { file, state };
  }

  return { logEvent, finalize, state };
}
