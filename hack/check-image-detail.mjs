#!/usr/bin/env node

import assert from "node:assert/strict";
import { pathToFileURL } from "node:url";

const DEFAULT_BASE_URL = "http://localhost:8088/api/v1";
const DEFAULT_TIMEOUT_MS = 10_000;

const scopes = [
  {
    key: "user",
    label: "current user",
    listPath: "images/kaniko/page",
    detailPath: "images/getbyid",
    tokenRole: "user",
  },
  {
    key: "admin",
    label: "administrator",
    listPath: "admin/images/kaniko/page",
    detailPath: "admin/images/getbyid",
    tokenRole: "admin",
  },
];

function requestURL(baseURL, path, params = {}) {
  const url = new URL(
    `${baseURL.replace(/\/$/, "")}/${path.replace(/^\//, "")}`,
  );
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== "") {
      url.searchParams.set(key, String(value));
    }
  }
  return url;
}

async function fetchResponse(url, token, timeoutMs) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(url, {
      method: "GET",
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      signal: controller.signal,
    });
  } finally {
    clearTimeout(timer);
  }
}

async function fetchJSON(baseURL, path, params, token, timeoutMs) {
  const url = requestURL(baseURL, path, params);
  const response = await fetchResponse(url, token, timeoutMs);
  if (!response.ok) {
    throw new Error(`${path} returned HTTP ${response.status}`);
  }
  return { url, body: await response.json() };
}

function readPage(body) {
  const page = body?.data;
  assert.ok(page && typeof page === "object", "response is missing data");
  assert.ok(Array.isArray(page.items), "data.items is not an array");
  for (const key of ["total", "page", "page_size"]) {
    assert.ok(
      Number.isSafeInteger(page[key]) && page[key] >= 0,
      `data.${key} is not a non-negative integer`,
    );
  }
  assert.equal(page.page, 1, "detail seed must come from page one");
  assert.ok(page.page_size > 0, "data.page_size must be positive");
  assert.ok(
    page.items.length <= page.page_size,
    "data.items exceeds data.page_size",
  );
  return page;
}

function readDetail(body, expectedID) {
  const detail = body?.data;
  assert.ok(detail && typeof detail === "object", "response is missing data");
  const actualID = detail.ID ?? detail.id;
  assert.equal(
    Number(actualID),
    Number(expectedID),
    "detail ID differs from list ID",
  );
  for (const key of [
    "imageLink",
    "status",
    "createdAt",
    "description",
    "dockerfile",
  ]) {
    assert.ok(
      Object.prototype.hasOwnProperty.call(detail, key),
      `detail.${key} is missing`,
    );
  }
  assert.equal(
    typeof detail.imageLink,
    "string",
    "detail.imageLink is not a string",
  );
  assert.equal(typeof detail.status, "string", "detail.status is not a string");
  assert.equal(
    typeof detail.createdAt,
    "string",
    "detail.createdAt is not a string",
  );
  return detail;
}

function itemID(item) {
  const id = item?.ID ?? item?.id;
  assert.ok(
    Number.isSafeInteger(Number(id)) && Number(id) > 0,
    "list item has no valid ID",
  );
  return Number(id);
}

async function checkScope(scope, options) {
  const { body } = await fetchJSON(
    options.baseURL,
    scope.listPath,
    { page: 1, page_size: 1, sort: "-createdAt" },
    options.token,
    options.timeoutMs,
  );
  const page = readPage(body);
  if (page.items.length === 0) {
    assert.equal(page.total, 0, `${scope.key} empty page has non-zero total`);
    return { total: 0, skipped: true };
  }

  const item = page.items[0];
  const id = itemID(item);
  const detailResponse = await fetchJSON(
    options.baseURL,
    scope.detailPath,
    { id },
    options.token,
    options.timeoutMs,
  );
  const detail = readDetail(detailResponse.body, id);
  if (item.imageLink !== undefined) {
    assert.equal(
      detail.imageLink,
      item.imageLink,
      "detail imageLink differs from list",
    );
  }
  if (item.status !== undefined) {
    assert.equal(detail.status, item.status, "detail status differs from list");
  }
  return { total: page.total, id, skipped: false };
}

export async function runImageDetailAcceptance({
  baseURL = process.env.CRATER_API_BASE_URL || DEFAULT_BASE_URL,
  userToken = process.env.CRATER_USER_AUTH_TOKEN ||
    process.env.CRATER_AUTH_TOKEN ||
    "",
  adminToken = process.env.CRATER_ADMIN_AUTH_TOKEN ||
    process.env.CRATER_AUTH_TOKEN ||
    "",
  timeoutMs = Number(
    process.env.CRATER_ACCEPTANCE_TIMEOUT_MS || DEFAULT_TIMEOUT_MS,
  ),
  log = console.log,
} = {}) {
  assert.ok(userToken, "set CRATER_USER_AUTH_TOKEN or CRATER_AUTH_TOKEN");
  assert.ok(adminToken, "set CRATER_ADMIN_AUTH_TOKEN or CRATER_AUTH_TOKEN");
  assert.ok(
    Number.isFinite(timeoutMs) && timeoutMs > 0,
    "timeout must be positive",
  );

  const unauthorized = await fetchResponse(
    requestURL(baseURL, scopes[0].listPath, { page: 1, page_size: 1 }),
    "",
    timeoutMs,
  );
  assert.equal(
    unauthorized.status,
    401,
    "unauthenticated image list must return HTTP 401",
  );

  const results = {};
  for (const scope of scopes) {
    const token = scope.tokenRole === "admin" ? adminToken : userToken;
    const result = await checkScope(scope, {
      baseURL,
      token,
      timeoutMs,
    });
    results[scope.key] = result;
    log(
      result.skipped
        ? `[PASS] ${scope.label}: no visible image builds`
        : `[PASS] ${scope.label}: total=${result.total}, detail_id=${result.id}`,
    );
  }
  assert.ok(
    !results.admin.skipped,
    "administrator has no image build data to verify",
  );
  log("[PASS] image build details match their paginated list records");
  return { ok: true, scopes: results };
}

function printHelp() {
  console.log(`Usage: CRATER_AUTH_TOKEN=... node hack/check-image-detail.mjs [options]

Runs read-only GET checks against paginated image-build lists and stable-ID detail
endpoints. Tokens are read from environment variables and are never printed.

Environment:
  CRATER_API_BASE_URL          API root (default: ${DEFAULT_BASE_URL})
  CRATER_AUTH_TOKEN            Token used for both user and admin scopes
  CRATER_USER_AUTH_TOKEN       Optional token for user scope
  CRATER_ADMIN_AUTH_TOKEN      Optional token for administrator scope
  CRATER_ACCEPTANCE_TIMEOUT_MS Per-request timeout (default: ${DEFAULT_TIMEOUT_MS})

Options:
  --base-url URL               Override the API root
  --timeout MS                 Override the per-request timeout
  -h, --help                   Show this help
`);
}

function parseArgs(argv) {
  const options = {};
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--base-url") options.baseURL = argv[++index];
    else if (arg === "--timeout") options.timeoutMs = Number(argv[++index]);
    else if (arg === "--help" || arg === "-h") options.help = true;
    else throw new Error(`unknown argument: ${arg}`);
  }
  return options;
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    printHelp();
    return;
  }
  await runImageDetailAcceptance(options);
}

const entrypoint =
  process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url;
if (entrypoint) {
  main().catch((error) => {
    console.error(`FAIL: ${error.message}`);
    process.exitCode = 1;
  });
}
