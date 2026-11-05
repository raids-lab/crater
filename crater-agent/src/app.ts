import { serve } from "@hono/node-server";
import { Hono } from "hono";

import { loadConfig } from "./config.js";
import { createChatRoutes } from "./routes/chat.js";

const config = loadConfig();
const app = new Hono();

app.get("/healthz", (context) => context.json({ status: "ok" }));
app.route("/", createChatRoutes(config));
app.onError((error, context) => {
  console.error("Crater-Agent request failed", error);
  return context.json({ error: "Crater-Agent request failed" }, 500);
});

serve({ fetch: app.fetch, port: config.port }, (info) => {
  console.info(`Crater-Agent is listening on port ${String(info.port)}`);
});
