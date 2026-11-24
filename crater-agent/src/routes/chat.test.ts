import assert from "node:assert/strict";
import { test } from "node:test";
import type { UIMessage } from "ai";

import { createChatRoutes } from "./chat.js";

void test("an explicit Jupyter creation request opens the form without relying on model tool choice", async (t) => {
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
      assert.ok(url.endsWith("/complete"), `unexpected model request: ${url}`);
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

void test("a later tool-enabled turn does not replay synthetic form tool calls to a thinking model", async (t) => {
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
        const body = JSON.parse(init.body) as {
          messages: { role: string; tool_calls?: unknown }[];
        };
        assert.ok(body.messages.every((message) => !message.tool_calls));
        assert.ok(
          body.messages.every((message) => message.role !== "assistant"),
        );
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

void test("a confirmed form navigation continues the existing assistant message", async (t) => {
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
        return Promise.resolve(
          Response.json({
            choices: [
              {
                index: 0,
                message: {
                  role: "assistant",
                  content: JSON.stringify({
                    text: "已准备表单建议。",
                    summary: "基础配置",
                    patches: [
                      {
                        path: "jobName",
                        value: "python-demo",
                        reason: "短名称",
                      },
                      {
                        path: "task.resource.cpu",
                        value: 4,
                        reason: "入门配置",
                      },
                      {
                        path: "task.resource.memory",
                        value: 8,
                        reason: "入门配置",
                      },
                      {
                        path: "task.resource.gpu.count",
                        value: 1,
                        reason: "试用 GPU",
                      },
                      {
                        path: "task.resource.gpu.model",
                        value: "nvidia.com/p100",
                        reason: "入门型号",
                      },
                      {
                        path: "task.image.imageLink",
                        value: "registry.example/python-basic:latest",
                        reason: "Python 基础镜像",
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
        input: { path: "/portal/jobs/new/jupyter-job" },
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
              text: "创建 Jupyter 作业跑 Python demo，申请一张入门 GPU",
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
  assert.match(await response.text(), /已准备表单建议/);
  assert.equal(completions.length, 1);
  const completion = completions[0];
  assert.ok(completion);
  assert.equal(completion.id, assistant.id);
  assert.ok(
    completion.parts.some((part) => part.type === "tool-navigate_to_page"),
  );
  assert.ok(
    completion.parts.some(
      (part) => part.type === "text" && part.text === "已准备表单建议。",
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
