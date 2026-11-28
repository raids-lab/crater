import assert from "node:assert/strict";
import { test } from "node:test";
import type { UIMessage } from "ai";

import { boundRunCompletion, createChatRoutes } from "./chat.js";

void test("query nodes before proposing both requested pages without form assistance", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  const events: string[] = [];
  const destinations = [
    {
      path: "/portal/jobs/new/jupyter-job",
      label: "New Jupyter job",
      reason: "Open the requested creation page without filling it",
      assistForm: false,
    },
    {
      path: "/portal/env/images",
      label: "Image list",
      reason: "Open the image list last",
      assistForm: false,
    },
  ];
  let modelCalls = 0;
  let saved: UIMessage | undefined;
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      if (url.endsWith("/llm/chat/completions")) {
        assert.ok(init && typeof init.body === "string");
        const first = modelCalls++ === 0;
        events.push(first ? "model-query" : "model-navigation");
        const body = JSON.parse(init.body) as {
          messages: unknown[];
          tools: unknown[];
        };
        assert.match(JSON.stringify(body.tools), /multiple requested pages/);
        if (!first) {
          assert.deepEqual(events, [
            "model-query",
            "nodes",
            "model-navigation",
          ]);
          assert.match(JSON.stringify(body.messages), /worker-amd64/);
        }
        const delta = first
          ? {
              tool_calls: [
                {
                  index: 0,
                  id: "nodes",
                  type: "function",
                  function: { name: "list_nodes", arguments: "{}" },
                },
              ],
            }
          : {
              content:
                "worker-amd64: amd64; worker-arm64: arm64. Confirm Jupyter first, then the image list. No form will be filled.",
              tool_calls: destinations.map((destination, index) => ({
                index,
                id: `navigation-${String(index)}`,
                type: "function",
                function: {
                  name: "navigate_to_page",
                  arguments: JSON.stringify(destination),
                },
              })),
            };
        return Promise.resolve(
          new Response(
            `data: ${JSON.stringify({ choices: [{ index: 0, delta, finish_reason: null }] })}\n\n` +
              `data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: "tool_calls" }] })}\n\n` +
              "data: [DONE]\n\n",
            { headers: { "Content-Type": "text/event-stream" } },
          ),
        );
      }
      if (url.endsWith("/api/agent/nodes")) {
        events.push("nodes");
        return Promise.resolve(
          Response.json({
            code: 0,
            data: [
              { name: "worker-amd64", architecture: "amd64" },
              { name: "worker-arm64", architecture: "arm64" },
            ],
          }),
        );
      }
      assert.ok(url.endsWith("/complete"));
      assert.ok(init && typeof init.body === "string");
      events.push("complete");
      const completion = JSON.parse(init.body) as {
        status: string;
        message: UIMessage;
      };
      assert.equal(completion.status, "completed");
      saved = completion.message;
      return Promise.resolve(Response.json({ code: 0 }));
    },
  );
  const response = await createChatRoutes(config).request("/chat", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${config.internalToken}`,
      "X-Crater-Agent-Context": "signed-test-context",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
      identityScope: "user",
      messages: [
        {
          id: "nodes-then-pages",
          role: "user",
          parts: [
            {
              type: "text",
              text: "先查询并列出集群节点的名称和架构，再带我进入新建 Jupyter 作业页面，暂时不要填写表单。最后跳转镜像列表页面。",
            },
          ],
        },
      ],
    }),
  });
  assert.equal(response.status, 200);
  const stream = await response.text();
  assert.match(stream, /worker-amd64/);
  assert.doesNotMatch(stream, /propose_form_patch|"type":"error"/);
  assert.deepEqual(events, [
    "model-query",
    "nodes",
    "model-navigation",
    "complete",
  ]);
  assert.ok(saved);
  const cards = saved.parts.filter(
    (part) => part.type === "tool-navigate_to_page",
  );
  assert.equal(cards.length, 2);
  assert.ok(
    cards.every((part) => "state" in part && part.state === "input-available"),
  );
  assert.deepEqual(
    cards.map((part) => ("input" in part ? part.input : undefined)),
    destinations,
  );
  assert.ok(
    saved.parts.findIndex((part) => part.type === "tool-list_nodes") <
      saved.parts.findIndex((part) => part.type === "text"),
  );
  assert.ok(
    saved.parts.findIndex((part) => part.type === "text") <
      saved.parts.findIndex((part) => part.type === "tool-navigate_to_page"),
  );
});

void test("whole-message budgeting preserves the answer and navigation but not oversized read evidence", () => {
  const message: UIMessage = {
    id: "large-answer",
    role: "assistant",
    parts: [
      { type: "text", text: "Node summary." },
      ...Array.from({ length: 8 }, (_, index) => ({
        type: "tool-list_nodes" as const,
        toolCallId: `nodes-${String(index)}`,
        state: "output-available" as const,
        input: {},
        output: [{ name: "worker", data: "x".repeat(48 * 1_024) }],
      })),
      {
        type: "tool-navigate_to_page",
        toolCallId: "nav",
        state: "input-available",
        input: { path: "/portal/jobs/new/jupyter-job", assistForm: true },
      },
    ],
  };
  const result = boundRunCompletion({ status: "completed", message });
  assert.equal(result.status, "completed");
  assert.deepEqual(result.message?.parts[0], message.parts[0]);
  assert.deepEqual(result.message?.parts.at(-1), message.parts.at(-1));
  assert.match(JSON.stringify(result), /truncated/);
  assert.ok(
    Buffer.byteLength(JSON.stringify(result.message?.parts)) < 240 * 1_024,
  );
  const failed = boundRunCompletion({
    status: "completed",
    message: {
      ...message,
      parts: [{ type: "text", text: "x".repeat(300_000) }],
    },
  });
  assert.equal(failed.status, "failed");
  assert.equal(failed.message, undefined);
});

for (const scenario of ["save-error", "too-large"] as const) {
  void test(`${scenario}: completion problems are reported in the stream`, async (t) => {
    const config = {
      port: 8000,
      backendInternalURL: "http://backend.invalid",
      internalToken: "isolated-test-internal-token-value",
    };
    let completionCalls = 0;
    t.mock.method(
      globalThis,
      "fetch",
      (input: string | URL | Request, init?: RequestInit) => {
        const url = input instanceof Request ? input.url : input.toString();
        if (url.endsWith("/complete")) {
          completionCalls++;
          if (scenario === "save-error")
            return Promise.resolve(Response.json({ code: 1 }, { status: 400 }));
          assert.ok(init && typeof init.body === "string");
          const body = JSON.parse(init.body) as {
            status: string;
            message?: unknown;
          };
          assert.equal(body.status, "failed");
          assert.equal(body.message, undefined);
          return Promise.resolve(Response.json({ code: 0 }));
        }
        assert.ok(url.endsWith("/llm/chat/completions"));
        return Promise.resolve(
          new Response(
            `data: ${JSON.stringify({ choices: [{ index: 0, delta: { content: scenario === "save-error" ? "Answer." : "x".repeat(300_000) }, finish_reason: null }] })}\n\n` +
              `data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: "stop" }] })}\n\n` +
              "data: [DONE]\n\n",
            { headers: { "Content-Type": "text/event-stream" } },
          ),
        );
      },
    );
    const response = await createChatRoutes(config).request("/chat", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${config.internalToken}`,
        "X-Crater-Agent-Context": "signed-test-context",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
        identityScope: "user",
        messages: [
          {
            id: "user",
            role: "user",
            parts: [{ type: "text", text: "Hello" }],
          },
        ],
      }),
    });
    assert.equal(response.status, 200);
    const stream = await response.text();
    assert.match(stream, /"type":"error"/);
    assert.match(
      stream,
      scenario === "save-error" ? /回答保存失败/ : /回答过长/,
    );
    assert.equal(completionCalls, 1);
  });
}

void test("empty form proposals succeed; navigation-only confirmations do not force form proposals", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  let proposalCalls = 0;
  let regularCalls = 0;
  const completions: { status: string; message: UIMessage }[] = [];
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      assert.ok(init && typeof init.body === "string");
      if (url.endsWith("/complete")) {
        completions.push(
          JSON.parse(init.body) as { status: string; message: UIMessage },
        );
        return Promise.resolve(Response.json({ code: 0 }));
      }
      assert.ok(url.endsWith("/llm/chat/completions"));
      const body = JSON.parse(init.body) as { stream?: boolean };
      if (body.stream) {
        regularCalls++;
        return Promise.resolve(
          new Response(
            `data: ${JSON.stringify({ choices: [{ index: 0, delta: { content: "Navigation only." }, finish_reason: null }] })}\n\n` +
              `data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: "stop" }] })}\n\n` +
              "data: [DONE]\n\n",
            { headers: { "Content-Type": "text/event-stream" } },
          ),
        );
      }
      proposalCalls++;
      return Promise.resolve(
        Response.json({
          choices: [
            {
              index: 0,
              message: {
                role: "assistant",
                content: JSON.stringify({
                  text: "No matching image; unchanged.",
                  summary: "No changes",
                  patches: [],
                }),
              },
              finish_reason: "stop",
            },
          ],
        }),
      );
    },
  );
  for (const assistForm of [true, false]) {
    const response = await createChatRoutes(config).request("/chat", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${config.internalToken}`,
        "X-Crater-Agent-Context": "signed-test-context",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
        identityScope: "user",
        messages: [
          {
            id: "user",
            role: "user",
            parts: [
              {
                type: "text",
                text: assistForm
                  ? "Use the missing image, leave everything else unchanged."
                  : "Open Jupyter; do not fill it.",
              },
            ],
          },
          {
            id: "assistant",
            role: "assistant",
            parts: [
              {
                type: "tool-list_nodes",
                toolCallId: "nodes",
                state: "output-available",
                input: {},
                output: [{ name: "worker" }],
              },
              {
                type: "tool-navigate_to_page",
                toolCallId: "nav",
                state: "output-available",
                input: { path: "/portal/jobs/new/jupyter-job", assistForm },
                output: {
                  status: "accepted",
                  path: "/portal/jobs/new/jupyter-job",
                },
              },
            ],
          },
        ],
        pageContext: {
          route: "/portal/jobs/new/jupyter-job",
          action: "assist-form",
          title: "Jupyter",
          prompt: "Help",
          form: {
            instanceId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
            capabilityId: "jupyter",
            schemaVersion: 1,
            title: "Jupyter",
            fields: [
              {
                path: "task.image.imageLink",
                label: "Image",
                valueType: "string",
              },
            ],
            currentValues: {},
            options: {},
          },
        },
      }),
    });
    assert.equal(response.status, 200);
    const stream = await response.text();
    assert.match(stream, assistForm ? /No matching image/ : /Navigation only/);
    assert.doesNotMatch(stream, /propose_form_patch/);
  }
  assert.equal(proposalCalls, 1);
  assert.equal(regularCalls, 1);
  assert.equal(completions.length, 2);
  assert.ok(
    completions.every((completion) => completion.status === "completed"),
  );
});

void test("an explicit Jupyter creation request can use the navigation tool", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  const completions: UIMessage[] = [];
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      if (url.endsWith("/llm/chat/completions")) {
        return Promise.resolve(
          new Response(
            `data: ${JSON.stringify({ choices: [{ index: 0, delta: { content: "I'll open the Jupyter form." }, finish_reason: null }] })}\n\n` +
              `data: ${JSON.stringify({ choices: [{ index: 0, delta: { tool_calls: [{ index: 0, id: "navigate-1", type: "function", function: { name: "navigate_to_page", arguments: JSON.stringify({ path: "/portal/jobs/new/jupyter-job", label: "Open Jupyter form", reason: "Prepare the requested job" }) } }] }, finish_reason: null }] })}\n\n` +
              `data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: "tool_calls" }] })}\n\n` +
              "data: [DONE]\n\n",
            { headers: { "Content-Type": "text/event-stream" } },
          ),
        );
      }
      assert.ok(url.endsWith("/complete"), `unexpected request: ${url}`);
      assert.ok(init && typeof init.body === "string");
      completions.push(
        (JSON.parse(init.body) as { message: UIMessage }).message,
      );
      return Promise.resolve(Response.json({ code: 0 }));
    },
  );

  const response = await createChatRoutes(config).request("/chat", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${config.internalToken}`,
      "X-Crater-Agent-Context": "signed-test-context",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
      identityScope: "user",
      messages: [
        {
          id: "create-jupyter",
          role: "user",
          parts: [
            {
              type: "text",
              text: "我第一次用 Crater，想创建一个 Jupyter 作业跑 Python demo，帮我选一张 GPU 和基础镜像",
            },
          ],
        },
      ],
    }),
  });
  assert.equal(response.status, 200);
  assert.match(await response.text(), /navigate_to_page/);
  assert.equal(completions.length, 1);
  assert.ok(
    completions[0]?.parts.some(
      (part) =>
        part.type === "tool-navigate_to_page" &&
        part.state === "input-available" &&
        typeof part.input === "object" &&
        part.input !== null &&
        "path" in part.input &&
        part.input.path === "/portal/jobs/new/jupyter-job",
    ),
  );
});

void test("a negated Jupyter request does not bypass the model with a form navigation", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  let modelCalls = 0;
  t.mock.method(globalThis, "fetch", (input: string | URL | Request) => {
    const url = input instanceof Request ? input.url : input.toString();
    if (url.endsWith("/llm/chat/completions")) {
      modelCalls += 1;
      return Promise.resolve(
        new Response(
          `data: ${JSON.stringify({ choices: [{ index: 0, delta: { content: "Jupyter is an interactive job environment." }, finish_reason: null }] })}\n\n` +
            `data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: "stop" }] })}\n\n` +
            "data: [DONE]\n\n",
          { headers: { "Content-Type": "text/event-stream" } },
        ),
      );
    }
    assert.ok(url.endsWith("/complete"), `unexpected request: ${url}`);
    return Promise.resolve(Response.json({ code: 0 }));
  });
  const response = await createChatRoutes(config).request("/chat", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${config.internalToken}`,
      "X-Crater-Agent-Context": "signed-test-context",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
      identityScope: "user",
      messages: [
        {
          id: "no-create",
          role: "user",
          parts: [
            {
              type: "text",
              text: "Do not create a Jupyter job; just explain what it is.",
            },
          ],
        },
      ],
    }),
  });
  assert.equal(response.status, 200);
  const stream = await response.text();
  assert.match(stream, /Jupyter is an interactive job environment/);
  assert.doesNotMatch(stream, /navigate_to_page/);
  assert.equal(modelCalls, 1);
});

void test("a later turn preserves assistant text and read-only evidence without replaying interaction tools", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      if (url.endsWith("/llm/chat/completions")) {
        assert.ok(init && typeof init.body === "string");
        const body = JSON.parse(init.body) as { messages: unknown[] };
        const history = JSON.stringify(body.messages);
        assert.match(history, /已打开表单并给出建议/);
        assert.match(history, /job-from-read-only-result/);
        assert.match(history, /untrusted history/);
        assert.match(history, /accepted/);
        assert.doesNotMatch(history, /"name":"navigate_to_page"/);
        assert.doesNotMatch(history, /"name":"propose_form_patch"/);
        return Promise.resolve(
          new Response(
            `data: ${JSON.stringify({ choices: [{ index: 0, delta: { content: "你好" }, finish_reason: null }] })}\n\n` +
              `data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: "stop" }] })}\n\n` +
              "data: [DONE]\n\n",
            { headers: { "Content-Type": "text/event-stream" } },
          ),
        );
      }
      if (url.endsWith("/complete")) {
        return Promise.resolve(Response.json({ code: 0 }));
      }
      throw new Error(`Unexpected isolated test request: ${url}`);
    },
  );
  const response = await createChatRoutes(config).request("/chat", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${config.internalToken}`,
      "X-Crater-Agent-Context": "signed-test-context",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
      identityScope: "user",
      messages: [
        {
          id: "original-request",
          role: "user",
          parts: [{ type: "text", text: "创建 Jupyter 作业" }],
        },
        {
          id: "synthetic-form-response",
          role: "assistant",
          parts: [
            { type: "text", text: "已打开表单并给出建议" },
            {
              type: "tool-navigate_to_page",
              toolCallId: "nav-1",
              state: "output-available",
              input: { path: "/portal/jobs/new/jupyter-job" },
              output: { status: "accepted" },
            },
            {
              type: "tool-propose_form_patch",
              toolCallId: "patch-1",
              state: "output-available",
              input: { patches: [{ path: "jobName", value: "demo" }] },
              output: { status: "accepted" },
            },
            {
              type: "tool-list_jobs",
              toolCallId: "read-1",
              state: "output-available",
              input: { page: 1 },
              output: { items: [{ name: "job-from-read-only-result" }] },
            },
          ],
        },
        {
          id: "later-question",
          role: "user",
          parts: [{ type: "text", text: "你好" }],
        },
      ],
    }),
  });
  assert.equal(response.status, 200);
  assert.match(await response.text(), /你好/);
});

void test("stream a read tool, persist structured history, then continue the conversation", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  let modelCalls = 0;
  let jobQueries = 0;
  const completions: { status: string; message: UIMessage }[] = [];
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      if (url.endsWith("/llm/chat/completions")) {
        const first = modelCalls++ === 0;
        const delta = first
          ? {
              tool_calls: [
                {
                  index: 0,
                  id: "list-call",
                  type: "function",
                  function: {
                    name: "list_jobs",
                    arguments: JSON.stringify({
                      page: 1,
                      pageSize: 5,
                      statuses: ["Failed"],
                    }),
                  },
                },
              ],
            }
          : { content: "There are no failed jobs in this account." };
        const chunks = [
          { choices: [{ index: 0, delta, finish_reason: null }] },
          {
            choices: [
              {
                index: 0,
                delta: {},
                finish_reason: first ? "tool_calls" : "stop",
              },
            ],
          },
        ];
        return Promise.resolve(
          new Response(
            chunks
              .map((chunk) => `data: ${JSON.stringify(chunk)}\n\n`)
              .join("") + "data: [DONE]\n\n",
            { headers: { "Content-Type": "text/event-stream" } },
          ),
        );
      }
      if (url.includes("/api/agent/jobs?")) {
        jobQueries++;
        assert.equal(new URL(url).searchParams.get("pageSize"), "5");
        return Promise.resolve(
          Response.json({ code: 0, data: { items: [], total: 0 } }),
        );
      }
      if (url.endsWith("/complete")) {
        assert.ok(init);
        assert.equal(typeof init.body, "string");
        completions.push(
          JSON.parse(init.body as string) as {
            status: string;
            message: UIMessage;
          },
        );
        return Promise.resolve(Response.json({ code: 0 }));
      }
      throw new Error(`Unexpected isolated test request: ${url}`);
    },
  );
  const routes = createChatRoutes(config);
  const user: UIMessage = {
    id: "question",
    role: "user",
    parts: [{ type: "text", text: "List only my five latest failed jobs" }],
  };
  const send = (messages: UIMessage[]) =>
    routes.request("/chat", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${config.internalToken}`,
        "X-Crater-Agent-Context": "signed-test-context",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
        identityScope: "user",
        messages,
      }),
    });
  const response = await send([user]);
  assert.equal(response.status, 200);
  const stream = await response.text();
  assert.match(stream, /There are no failed jobs/);
  assert.equal(jobQueries, 1);
  assert.equal(completions[0]?.status, "completed");
  const assistant = completions[0].message;
  assert.ok(assistant);
  assert.ok(
    assistant.parts.some(
      (part) =>
        part.type === "tool-list_jobs" && part.state === "output-available",
    ),
  );
  const followup = await send([
    user,
    assistant,
    {
      id: "followup",
      role: "user",
      parts: [{ type: "text", text: "Explain that briefly" }],
    },
  ]);
  assert.equal(followup.status, 200);
  assert.match(await followup.text(), /There are no failed jobs/);
  assert.equal(completions[1]?.status, "completed");
});

void test("a confirmed form navigation requests suggestions in the user's language", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  const completions: UIMessage[] = [];
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      if (url.endsWith("/llm/chat/completions")) {
        assert.ok(init && typeof init.body === "string");
        const body = JSON.parse(init.body) as {
          messages?: { role: string; tool_calls?: unknown }[];
          tool_choice?: unknown;
          tools?: { function?: { name?: string } }[];
        };
        assert.equal(body.tool_choice, undefined);
        assert.equal(body.tools, undefined);
        assert.ok(body.messages?.every((message) => !message.tool_calls));
        assert.match(
          JSON.stringify(body.messages),
          /用户最新一条消息的主要语言/,
        );
        assert.doesNotMatch(JSON.stringify(body.messages), /简短中文说明/);
        return Promise.resolve(
          Response.json({
            choices: [
              {
                index: 0,
                message: {
                  role: "assistant",
                  content: JSON.stringify({
                    text: "Form suggestions are ready.",
                    summary: "Starter setup",
                    patches: [
                      {
                        path: "jobName",
                        value: "python-demo",
                        reason: "Short name",
                      },
                      {
                        path: "task.resource.cpu",
                        value: 4,
                        reason: "Starter configuration",
                      },
                      {
                        path: "task.resource.memory",
                        value: 8,
                        reason: "Starter configuration",
                      },
                      {
                        path: "task.resource.gpu.count",
                        value: 1,
                        reason: "Try a GPU",
                      },
                      {
                        path: "task.resource.gpu.model",
                        value: "nvidia.com/p100",
                        reason: "Entry-level model",
                      },
                      {
                        path: "task.image.imageLink",
                        value: "registry.example/python-basic:latest",
                        reason: "Basic Python image",
                      },
                    ],
                  }),
                },
                finish_reason: "stop",
              },
            ],
          }),
        );
      }
      if (url.endsWith("/complete")) {
        assert.ok(init && typeof init.body === "string");
        const completion = JSON.parse(init.body) as {
          message: UIMessage;
        };
        completions.push(completion.message);
        return Promise.resolve(Response.json({ code: 0 }));
      }
      throw new Error(`Unexpected isolated test request: ${url}`);
    },
  );
  const assistant: UIMessage = {
    id: "assistant-nav",
    role: "assistant",
    parts: [
      { type: "text", text: "打开表单。" },
      {
        type: "tool-navigate_to_page",
        toolCallId: "nav-1",
        state: "output-available",
        input: { path: "/portal/jobs/new/jupyter-job", assistForm: true },
        output: { status: "accepted", path: "/portal/jobs/new/jupyter-job" },
      },
    ],
  };
  const response = await createChatRoutes(config).request("/chat", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${config.internalToken}`,
      "X-Crater-Agent-Context": "signed-test-context",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
      identityScope: "user",
      messages: [
        {
          id: "user-nav",
          role: "user",
          parts: [
            {
              type: "text",
              text: "Create a Jupyter job for a Python demo with one entry-level GPU",
            },
          ],
        },
        assistant,
      ],
      pageContext: {
        route: "/portal/jobs/new/jupyter-job",
        action: "assist-form",
        title: "Jupyter",
        prompt: "填写表单",
        form: {
          instanceId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
          capabilityId: "jupyter",
          schemaVersion: 1,
          title: "Jupyter",
          fields: [
            { path: "jobName", label: "名称", valueType: "string" },
            { path: "task.resource.cpu", label: "CPU", valueType: "number" },
            {
              path: "task.resource.memory",
              label: "内存",
              valueType: "number",
            },
            {
              path: "task.resource.gpu.count",
              label: "GPU 数量",
              valueType: "number",
            },
            {
              path: "task.resource.gpu.model",
              label: "GPU 型号",
              valueType: "string",
            },
            {
              path: "task.image.imageLink",
              label: "镜像",
              valueType: "string",
            },
          ],
          currentValues: {},
          options: {
            "task.resource.gpu.model": {
              items: [
                { value: "nvidia.com/p100", label: "P100" },
                { value: "nvidia.com/v100", label: "V100" },
              ],
              total: 2,
            },
            "task.image.imageLink": {
              items: [
                {
                  value: "registry.example/python-basic:latest",
                  label: "Python basic",
                },
              ],
              total: 1,
            },
          },
        },
      },
    }),
  });
  assert.equal(response.status, 200);
  assert.match(await response.text(), /Form suggestions are ready/);
  assert.equal(completions.length, 1);
  const completion = completions[0];
  assert.ok(completion);
  assert.equal(completion.id, assistant.id);
  assert.ok(
    completion.parts.some((part) => part.type === "tool-navigate_to_page"),
  );
  assert.ok(
    completion.parts.some(
      (part) =>
        part.type === "text" && part.text === "Form suggestions are ready.",
    ),
  );
  assert.ok(
    completion.parts.some(
      (part) =>
        part.type === "tool-propose_form_patch" &&
        part.state === "input-available" &&
        typeof part.input === "object" &&
        part.input !== null &&
        "patches" in part.input &&
        Array.isArray(part.input.patches) &&
        part.input.patches.length === 6,
    ),
  );
});

void test("a new owner job-list question does not reuse an old failed lookup", async (t) => {
  const config = {
    port: 8000,
    backendInternalURL: "http://backend.invalid",
    internalToken: "isolated-test-internal-token-value",
  };
  const toolChoices: unknown[] = [];
  const usernames: string[] = [];
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      if (url.endsWith("/llm/chat/completions")) {
        const bodyText = init?.body;
        assert.ok(typeof bodyText === "string");
        const body = JSON.parse(bodyText) as {
          tool_choice?: unknown;
        };
        toolChoices.push(body.tool_choice);
        if (typeof body.tool_choice === "object") {
          return Promise.resolve(
            Response.json(
              { error: { message: "named tool choice is unsupported" } },
              { status: 400 },
            ),
          );
        }
        const first = toolChoices.length === 1;
        const delta = first
          ? {
              tool_calls: [
                {
                  index: 0,
                  id: "fresh-list",
                  type: "function",
                  function: {
                    name: "list_jobs",
                    arguments: JSON.stringify({
                      page: 1,
                      pageSize: 5,
                      username: "周霄",
                    }),
                  },
                },
              ],
            }
          : { content: "周霄的作业已查询。" };
        return Promise.resolve(
          new Response(
            `data: ${JSON.stringify({ choices: [{ index: 0, delta, finish_reason: null }] })}\n\n` +
              `data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: first ? "tool_calls" : "stop" }] })}\n\n` +
              "data: [DONE]\n\n",
            { headers: { "Content-Type": "text/event-stream" } },
          ),
        );
      }
      if (url.includes("/api/agent/jobs?")) {
        usernames.push(new URL(url).searchParams.get("username") ?? "");
        return Promise.resolve(
          Response.json({ code: 0, data: { items: [], hasMore: false } }),
        );
      }
      if (url.endsWith("/complete"))
        return Promise.resolve(Response.json({ code: 0 }));
      throw new Error(`Unexpected isolated test request: ${url}`);
    },
  );

  const messages: UIMessage[] = [
    {
      id: "old-user",
      role: "user",
      parts: [{ type: "text", text: "周霄有哪些作业" }],
    },
    {
      id: "old-assistant",
      role: "assistant",
      parts: [
        {
          type: "tool-list_jobs",
          toolCallId: "old-list",
          state: "output-error",
          input: { username: "周霄" },
          errorText: "service unavailable",
        },
        { type: "text", text: "暂时查不到。" },
      ],
    },
    {
      id: "new-user",
      role: "user",
      parts: [{ type: "text", text: "周霄有哪些作业" }],
    },
  ];
  const response = await createChatRoutes(config).request("/chat", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${config.internalToken}`,
      "X-Crater-Agent-Context": "signed-test-context",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      runId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
      identityScope: "admin",
      messages,
    }),
  });
  assert.equal(response.status, 200);
  assert.match(await response.text(), /周霄的作业已查询/);
  assert.deepEqual(usernames, ["周霄"]);
  assert.equal(toolChoices[0], "auto");
  assert.equal(toolChoices[1], "auto");
});
