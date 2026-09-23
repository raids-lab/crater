import assert from "node:assert/strict";
import { createServer } from "node:http";
import { after, before, describe, it } from "node:test";

import { runEMIASAcceptance } from "./check-emias-pagination.mjs";

const selfItems = [
  {
    id: 1,
    name: "train-alpha",
    owner: "alice",
    queue: "team",
    status: "Completed",
    creationTimestamp: "2026-09-01T01:00:00Z",
  },
  {
    id: 2,
    name: "train-beta",
    owner: "alice",
    queue: "team",
    status: "Running",
    creationTimestamp: "2026-09-02T01:00:00Z",
  },
  {
    id: 3,
    name: "debug-gamma",
    owner: "alice",
    queue: "team",
    status: "Completed",
    creationTimestamp: "2026-09-03T01:00:00Z",
  },
];
const accountItems = [
  ...selfItems,
  {
    id: 4,
    name: "train-delta",
    owner: "bob",
    queue: "team",
    status: "Failed",
    creationTimestamp: "2026-09-04T01:00:00Z",
  },
];
const adminItems = [
  ...accountItems,
  {
    id: 5,
    name: "train-epsilon",
    owner: "carol",
    queue: "other",
    status: "Pending",
    creationTimestamp: "2026-09-05T01:00:00Z",
  },
];

let server;
let baseURL;
const methods = [];

function itemsForPath(pathname) {
  if (pathname.startsWith("/api/v1/admin/aijobs/")) return adminItems;
  if (pathname.startsWith("/api/v1/aijobs/all/")) return accountItems;
  return selfItems;
}

function filteredItems(url) {
  let items = [...itemsForPath(url.pathname)];
  const search = url.searchParams.get("search")?.toLocaleLowerCase();
  if (search) {
    items = items.filter((item) =>
      [item.name, item.owner, item.queue].some((value) =>
        value.toLocaleLowerCase().includes(search),
      ),
    );
  }
  const statuses = url.searchParams.getAll("status");
  if (statuses.length > 0)
    items = items.filter((item) => statuses.includes(item.status));
  const descending = url.searchParams.get("sort") !== "createdAt";
  items.sort((left, right) =>
    descending
      ? Date.parse(right.creationTimestamp) - Date.parse(left.creationTimestamp)
      : Date.parse(left.creationTimestamp) -
        Date.parse(right.creationTimestamp),
  );
  return items;
}

before(async () => {
  server = createServer((request, response) => {
    methods.push(request.method);
    const url = new URL(request.url, "http://localhost");
    if (!request.headers.authorization) {
      response.writeHead(401, { "content-type": "application/json" });
      response.end(JSON.stringify({ message: "unauthorized" }));
      return;
    }

    const expectedToken = url.pathname.startsWith("/api/v1/admin/")
      ? "Bearer admin-token"
      : "Bearer user-token";
    if (request.headers.authorization !== expectedToken) {
      response.writeHead(403, { "content-type": "application/json" });
      response.end(JSON.stringify({ message: "forbidden" }));
      return;
    }

    const items = filteredItems(url);
    response.setHeader("content-type", "application/json");
    if (url.pathname.endsWith("/facets")) {
      const counts = new Map();
      for (const item of items)
        counts.set(item.status, (counts.get(item.status) || 0) + 1);
      response.end(
        JSON.stringify({
          code: 0,
          data: {
            facets: {
              job_type: [],
              priority: [],
              profile_status: [],
              status: [...counts].map(([value, count]) => ({ value, count })),
            },
          },
        }),
      );
      return;
    }

    const page = Number(url.searchParams.get("page"));
    const pageSize = Number(url.searchParams.get("page_size"));
    const offset = (page - 1) * pageSize;
    response.end(
      JSON.stringify({
        code: 0,
        data: {
          items:
            offset >= items.length
              ? []
              : items.slice(offset, offset + pageSize),
          total: items.length,
          page,
          page_size: pageSize,
        },
      }),
    );
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  baseURL = `http://127.0.0.1:${server.address().port}/api/v1`;
});

after(async () => {
  await new Promise((resolve, reject) =>
    server.close((error) => (error ? reject(error) : resolve())),
  );
});

describe("EMIAS real-environment acceptance runner", () => {
  it("checks all scopes with read-only requests and no token output", async () => {
    const logs = [];
    const result = await runEMIASAcceptance({
      baseURL,
      userToken: "user-token",
      adminToken: "admin-token",
      timeoutMs: 1000,
      log: (message) => logs.push(message),
    });

    assert.equal(result.ok, true);
    assert.deepEqual(
      Object.fromEntries(
        Object.entries(result.scopes).map(([key, value]) => [key, value.total]),
      ),
      { self: 3, account: 4, admin: 5 },
    );
    assert.ok(methods.every((method) => method === "GET"));
    assert.ok(
      logs.every(
        (line) => !line.includes("user-token") && !line.includes("admin-token"),
      ),
    );
  });
});
