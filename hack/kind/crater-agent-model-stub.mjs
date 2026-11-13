// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

import { createServer } from "node:http";

// Only the isolated Kind smoke test uses this deterministic OpenAI-compatible model.
createServer(async (request, response) => {
  if (request.method !== "POST" || request.url !== "/v1/chat/completions") {
    response.writeHead(404).end();
    return;
  }
  let body = "";
  for await (const chunk of request) body += chunk;
  const { model, stream } = JSON.parse(body);
  const completion = {
    id: "crater-agent-smoke-completion",
    created: 0,
    model,
  };
  const content = "Crater-Agent deterministic smoke response";
  if (!stream) {
    response.writeHead(200, { "Content-Type": "application/json" });
    response.end(
      JSON.stringify({
        ...completion,
        object: "chat.completion",
        choices: [
          {
            index: 0,
            message: { role: "assistant", content },
            finish_reason: "stop",
          },
        ],
      }),
    );
    return;
  }
  response.writeHead(200, { "Content-Type": "text/event-stream" });
  for (const choice of [
    { index: 0, delta: { role: "assistant", content }, finish_reason: null },
    { index: 0, delta: {}, finish_reason: "stop" },
  ]) {
    response.write(
      `data: ${JSON.stringify({ ...completion, object: "chat.completion.chunk", choices: [choice] })}\n\n`,
    );
  }
  response.end("data: [DONE]\n\n");
}).listen(8000, "0.0.0.0");
