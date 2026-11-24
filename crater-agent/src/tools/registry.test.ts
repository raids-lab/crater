import assert from "node:assert/strict";
import { test } from "node:test";
import { convertToModelMessages, type ToolExecutionOptions } from "ai";

import { selectSkill } from "../skills/select.js";
import { pageContextSchema } from "../page-context.js";
import { selectSkillTools } from "./registry.js";

const config = {
  port: 8000,
  backendInternalURL: "http://backend.invalid",
  internalToken: "test-token-not-a-real-secret-value",
};
const options: ToolExecutionOptions<unknown> = {
  toolCallId: "test",
  messages: [],
  context: undefined,
};

void test("job lists skip exact counts unless requested", async (t) => {
  const urls: string[] = [];
  t.mock.method(globalThis, "fetch", (input: string | URL | Request) => {
    urls.push(input instanceof Request ? input.url : input.toString());
    return Promise.resolve(
      Response.json({ code: 0, data: { items: [], hasMore: false } }),
    );
  });
  const tools = selectSkillTools(
    selectSkill("user").activeTools,
    config,
    "signed",
    "user",
  );
  const execute = tools.list_jobs?.execute;
  assert.ok(execute);
  await execute({ page: 1, pageSize: 5, includeTotal: false }, options);
  await execute({ page: 1, pageSize: 5, includeTotal: true }, options);
  assert.equal(urls.length, 2);
  const first = urls[0];
  const second = urls[1];
  assert.ok(first);
  assert.ok(second);
  assert.equal(new URL(first).searchParams.has("includeTotal"), false);
  assert.equal(new URL(second).searchParams.get("includeTotal"), "true");
});

void test("all entry points share read tools; navigation stays role scoped", async () => {
  const user = selectSkill("user");
  const admin = selectSkill("admin");
  assert.deepEqual(user.activeTools, admin.activeTools);
  const context = pageContextSchema.parse({
    route: "/portal/jobs/new/jupyter-job",
    action: "assist-form",
    title: "Form",
    prompt: "Help",
    form: {
      instanceId: "9ed49cd2-c1be-4f55-8985-032cd72b4865",
      capabilityId: "jupyter",
      schemaVersion: 1,
      title: "Jupyter",
      fields: [{ path: "jobName", label: "Name", valueType: "string" }],
      currentValues: {},
    },
  });
  const form = selectSkill("user", context);
  for (const name of user.activeTools.filter(
    (tool) => tool !== "navigate_to_page",
  ))
    assert.ok(form.activeTools.includes(name));
  assert.ok(!form.activeTools.includes("navigate_to_page"));
  assert.ok(form.activeTools.includes("propose_form_patch"));
  const tools = selectSkillTools(admin.activeTools, config, "signed", "admin");
  const schema = tools.navigate_to_page?.inputSchema;
  assert.ok(schema && "safeParse" in schema);
  const parse = schema as {
    safeParse: (value: unknown) => { success: boolean };
  };
  assert.equal(
    parse.safeParse({
      path: "/portal/jobs/inter",
      label: "Jobs",
      reason: "Requested",
    }).success,
    false,
  );
  assert.equal(
    parse.safeParse({ path: "/admin/jobs", label: "Jobs", reason: "Requested" })
      .success,
    true,
  );
  const messages = await convertToModelMessages(
    [
      {
        role: "assistant",
        parts: [
          {
            type: "tool-navigate_to_page",
            toolCallId: "partial",
            state: "input-streaming",
          },
        ],
      },
    ],
    { ignoreIncompleteToolCalls: true },
  );
  assert.ok(!JSON.stringify(messages).includes("partial"));
});

void test("bounded batch queries preserve partial errors and stop on cancellation", async (t) => {
  const calls: string[] = [];
  const controller = new AbortController();
  t.mock.method(
    globalThis,
    "fetch",
    (input: string | URL | Request, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : input.toString();
      calls.push(url);
      assert.equal(
        new Headers(init?.headers).get("X-Crater-Agent-Context"),
        "signed",
      );
      if (url.includes("cancel")) controller.abort();
      return Promise.resolve(
        Response.json({
          code: url.includes("missing") ? 404 : 0,
          data: {},
          msg: "not found",
        }),
      );
    },
  );
  const tools = selectSkillTools(
    selectSkill("user").activeTools,
    config,
    "signed",
    "user",
  );
  const execute = tools.get_job_diagnostic_snapshots?.execute;
  assert.ok(execute);
  const result: unknown = await execute(
    { jobNames: ["first", "first", "missing"] },
    options,
  );
  assert.equal(calls.length, 2);
  assert.deepEqual((result as { failed: number }).failed, 1);
  calls.length = 0;
  await assert.rejects(async () => {
    await execute(
      { jobNames: ["cancel", "never"] },
      { ...options, abortSignal: controller.signal },
    );
  });
  assert.equal(calls.length, 1);
});
