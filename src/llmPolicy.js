const fallbackFlow = [
  "browse_feed",
  "search",
  "like_top_post",
  "comment_top_post",
  "browse_feed",
  "create_post",
  "open_profile",
];

function nextFromFallback(stepIndex) {
  return fallbackFlow[stepIndex % fallbackFlow.length];
}

function sanitizeAction(raw) {
  const allowed = new Set([
    "login",
    "browse_feed",
    "search",
    "create_post",
    "like_top_post",
    "comment_top_post",
    "open_profile",
  ]);
  if (allowed.has(raw)) return raw;
  return "browse_feed";
}

export async function chooseNextAction({ config, account, stepIndex, recentActions }) {
  const noLlm = !config.llm.apiKey;
  if (noLlm) {
    return {
      action: nextFromFallback(stepIndex),
      reason: "fallback-policy",
      payload: {},
    };
  }

  const prompt = {
    role: "You are a test-user behavior planner for a social web app in a staging environment.",
    objective: "Choose one next action for realistic product testing coverage.",
    account: { id: account.id, username: account.username },
    recentActions,
    allowedActions: [
      "browse_feed",
      "search",
      "create_post",
      "like_top_post",
      "comment_top_post",
      "open_profile",
    ],
    constraints: [
      "Return strict JSON only.",
      "Do not include harmful or policy-violating content.",
      "Prefer diverse actions across steps.",
    ],
    responseSchema: {
      action: "one allowed action",
      payload: {
        searchQuery: "optional string",
        postText: "optional string",
        commentText: "optional string",
      },
      reason: "short reason",
    },
  };

  const resp = await fetch(`${config.llm.baseUrl}/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${config.llm.apiKey}`,
    },
    body: JSON.stringify({
      model: config.llm.model,
      temperature: 0.7,
      response_format: { type: "json_object" },
      messages: [
        {
          role: "system",
          content: "You are a deterministic test action planner.",
        },
        {
          role: "user",
          content: JSON.stringify(prompt),
        },
      ],
    }),
  });

  if (!resp.ok) {
    return {
      action: nextFromFallback(stepIndex),
      reason: `fallback-http-${resp.status}`,
      payload: {},
    };
  }

  const data = await resp.json();
  const content = data?.choices?.[0]?.message?.content;
  if (!content) {
    return {
      action: nextFromFallback(stepIndex),
      reason: "fallback-empty-llm",
      payload: {},
    };
  }

  try {
    const parsed = JSON.parse(content);
    return {
      action: sanitizeAction(parsed.action),
      reason: parsed.reason || "llm",
      payload: parsed.payload || {},
    };
  } catch {
    return {
      action: nextFromFallback(stepIndex),
      reason: "fallback-invalid-json",
      payload: {},
    };
  }
}
