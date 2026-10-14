import assert from "node:assert/strict";
import { createServer } from "node:http";
import { after, before, describe, it } from "node:test";

import { runImageDetailAcceptance } from "./check-image-detail.mjs";

const userItems = [
  {
    ID: 7,
    imageLink: "registry.example/train:latest",
    imagepackName: "train",
    status: "Finished",
    createdAt: "2026-09-01T01:00:00Z",
  },
];
const adminItems = [
  ...userItems,
  {
    ID: 8,
    imageLink: "registry.example/eval:latest",
    imagepackName: "eval",
    status: "Failed",
    createdAt: "2026-09-02T01:00:00Z",
  },
];

let server;
let baseURL;
const methods = [];

function itemsForPath(pathname) {
  return pathname.startsWith("/api/v1/admin/") ? adminItems : userItems;
}

before(async () => {
  server = createServer((request, response) => {
    methods.push(request.method);
    const url = new URL(request.url, "http://localhost");
    const isAdmin = url.pathname.startsWith("/api/v1/admin/");
    const expectedToken = isAdmin ? "Bearer admin-token" : "Bearer user-token";
    if (!request.headers.authorization) {
      response.writeHead(401, { "content-type": "application/json" });
      response.end(JSON.stringify({ message: "unauthorized" }));
      return;
    }
    if (request.headers.authorization !== expectedToken) {
      response.writeHead(403, { "content-type": "application/json" });
      response.end(JSON.stringify({ message: "forbidden" }));
      return;
    }

    const items = itemsForPath(url.pathname);
    response.setHeader("content-type", "application/json");
    if (url.pathname.endsWith("/page")) {
      const page = Number(url.searchParams.get("page"));
      const pageSize = Number(url.searchParams.get("page_size"));
      const offset = (page - 1) * pageSize;
      response.end(
        JSON.stringify({
          code: 0,
          data: {
            items: items.slice(offset, offset + pageSize),
            total: items.length,
            page,
            page_size: pageSize,
          },
        }),
      );
      return;
    }
    const id = Number(url.searchParams.get("id"));
    const item = items.find((candidate) => candidate.ID === id);
    if (!item) {
      response.writeHead(404, { "content-type": "application/json" });
      response.end(JSON.stringify({ message: "not found" }));
      return;
    }
    response.end(
      JSON.stringify({
        code: 0,
        data: {
          ID: item.ID,
          imageLink: item.imageLink,
          status: item.status,
          createdAt: item.createdAt,
          description: "",
          dockerfile: "",
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

describe("image detail acceptance runner", () => {
  it("checks stable-ID details without writing data or printing tokens", async () => {
    const logs = [];
    const result = await runImageDetailAcceptance({
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
      { user: 1, admin: 2 },
    );
    assert.ok(methods.every((method) => method === "GET"));
    assert.ok(
      logs.every(
        (line) => !line.includes("user-token") && !line.includes("admin-token"),
      ),
    );
  });
});
