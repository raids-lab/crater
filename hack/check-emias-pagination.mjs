#!/usr/bin/env node

import assert from "node:assert/strict";
import { pathToFileURL } from "node:url";

const DEFAULT_BASE_URL = "http://localhost:8088/api/v1";
const DEFAULT_TIMEOUT_MS = 10_000;
const PAGE_SIZE = 2;
const MAX_PAGE_SIZE = 200;
const OUT_OF_RANGE_PAGE = 999_999;

const scopes = [
  {
    key: "self",
    label: "current user",
    listPath: "aijobs/page",
    facetsPath: "aijobs/page/facets",
    tokenRole: "user",
  },
  {
    key: "account",
    label: "current account",
    listPath: "aijobs/all/page",
    facetsPath: "aijobs/all/page/facets",
    tokenRole: "user",
  },
  {
    key: "admin",
    label: "administrator",
    listPath: "admin/aijobs/page",
    facetsPath: "admin/aijobs/page/facets",
    tokenRole: "admin",
  },
];

function requestURL(baseURL, path, params = {}) {
  const url = new URL(
    `${baseURL.replace(/\/$/, "")}/${path.replace(/^\//, "")}`,
  );
  for (const [key, value] of Object.entries(params)) {
    const values = Array.isArray(value) ? value : [value];
    for (const item of values) {
      if (item !== undefined && item !== null && item !== "") {
        url.searchParams.append(key, String(item));
      }
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
  const body = await response.json();
  return { url, body };
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
  assert.ok(page.page_size > 0, "data.page_size must be positive");
  assert.ok(
    page.items.length <= page.page_size,
    "data.items exceeds data.page_size",
  );
  return page;
}

function readFacets(body) {
  const facets = body?.data?.facets;
  assert.ok(
    facets && typeof facets === "object" && !Array.isArray(facets),
    "response is missing data.facets",
  );
  for (const key of ["job_type", "priority", "profile_status", "status"]) {
    assert.ok(Array.isArray(facets[key]), `facets.${key} is not an array`);
  }
  for (const [key, values] of Object.entries(facets)) {
    for (const value of values) {
      assert.ok(value?.value !== undefined, `facets.${key} has no value`);
      assert.ok(
        Number.isSafeInteger(value?.count) && value.count >= 0,
        `facets.${key} has an invalid count`,
      );
    }
  }
  return facets;
}

function itemID(item) {
  for (const key of ["id", "jobName", "name"]) {
    if (item?.[key] !== undefined && item[key] !== null) {
      return `${key}:${String(item[key])}`;
    }
  }
  throw new Error("AIJob item has no stable identity");
}

function itemTimestamp(item) {
  const raw = item?.creationTimestamp ?? item?.createdAt;
  const value = Date.parse(raw);
  return Number.isFinite(value) ? value : undefined;
}

function assertSorted(items, descending) {
  for (let index = 1; index < items.length; index += 1) {
    const previous = itemTimestamp(items[index - 1]);
    const current = itemTimestamp(items[index]);
    if (previous === undefined || current === undefined) continue;
    if (descending) {
      assert.ok(
        previous >= current,
        "items are not sorted by descending creation time",
      );
    } else {
      assert.ok(
        previous <= current,
        "items are not sorted by ascending creation time",
      );
    }
  }
}

function assertNoOverlap(first, second) {
  const firstIDs = new Set(first.map(itemID));
  const duplicates = second.map(itemID).filter((id) => firstIDs.has(id));
  assert.deepEqual(duplicates, [], `pages overlap: ${duplicates.join(", ")}`);
}

async function getPage(scope, options, params) {
  const { body } = await fetchJSON(
    options.baseURL,
    scope.listPath,
    params,
    options.token,
    options.timeoutMs,
  );
  return readPage(body);
}

async function getFacets(scope, options, params = {}) {
  const { body } = await fetchJSON(
    options.baseURL,
    scope.facetsPath,
    params,
    options.token,
    options.timeoutMs,
  );
  return readFacets(body);
}

async function getAllItems(scope, options, expectedTotal) {
  const items = [];
  const pages = Math.max(1, Math.ceil(expectedTotal / MAX_PAGE_SIZE));
  for (let pageNumber = 1; pageNumber <= pages; pageNumber += 1) {
    const page = await getPage(scope, options, {
      page: pageNumber,
      page_size: MAX_PAGE_SIZE,
      sort: "-createdAt",
    });
    assert.equal(
      page.total,
      expectedTotal,
      `${scope.key} total changed while collecting items`,
    );
    items.push(...page.items);
  }
  assert.equal(
    items.length,
    expectedTotal,
    `${scope.key} did not return all items`,
  );
  assert.equal(
    new Set(items.map(itemID)).size,
    items.length,
    `${scope.key} contains duplicate items`,
  );
  return items;
}

function searchableText(item) {
  return [
    item?.name,
    item?.jobName,
    item?.owner,
    item?.queue,
    item?.userInfo?.username,
    item?.userInfo?.nickname,
  ]
    .filter(Boolean)
    .join("\n")
    .toLocaleLowerCase();
}

async function checkScope(scope, options) {
  const descending = await getPage(scope, options, {
    page: 1,
    page_size: PAGE_SIZE,
    sort: "-createdAt",
  });
  assert.ok(descending.total > 0, `${scope.key} has no real AIJob data`);
  assert.equal(descending.page, 1);
  assert.equal(descending.page_size, PAGE_SIZE);
  assertSorted(descending.items, true);

  const second = await getPage(scope, options, {
    page: 2,
    page_size: PAGE_SIZE,
    sort: "-createdAt",
  });
  assert.equal(
    second.total,
    descending.total,
    `${scope.key} total changed across pages`,
  );
  assertNoOverlap(descending.items, second.items);

  const resized = await getPage(scope, options, {
    page: 1,
    page_size: 1,
    sort: "-createdAt",
  });
  assert.equal(
    resized.total,
    descending.total,
    `${scope.key} total changed with page size`,
  );
  assert.equal(resized.items.length, 1);

  const ascending = await getPage(scope, options, {
    page: 1,
    page_size: PAGE_SIZE,
    sort: "createdAt",
  });
  assert.equal(
    ascending.total,
    descending.total,
    `${scope.key} total changed with sort`,
  );
  assertSorted(ascending.items, false);

  const outOfRange = await getPage(scope, options, {
    page: OUT_OF_RANGE_PAGE,
    page_size: PAGE_SIZE,
    sort: "-createdAt",
  });
  assert.equal(
    outOfRange.total,
    descending.total,
    `${scope.key} total changed out of range`,
  );
  assert.deepEqual(
    outOfRange.items,
    [],
    `${scope.key} out-of-range page is not empty`,
  );

  await getFacets(scope, options);
  const allItems = await getAllItems(scope, options, descending.total);
  const seed = allItems[0];
  const search = String(seed.name || seed.owner || seed.queue || seed.jobName);
  const searched = await getPage(scope, options, {
    page: 1,
    page_size: MAX_PAGE_SIZE,
    search,
    sort: "-createdAt",
  });
  assert.ok(
    searched.total > 0,
    `${scope.key} real-name search returned no items`,
  );
  assert.ok(
    searched.total <= descending.total,
    `${scope.key} search increased total`,
  );
  assert.ok(
    searched.items.every((item) =>
      searchableText(item).includes(search.toLocaleLowerCase()),
    ),
    `${scope.key} search returned a non-matching item`,
  );

  assert.ok(
    seed.status !== undefined && seed.status !== null && seed.status !== "",
    `${scope.key} seed item has no status`,
  );
  const status = String(seed.status);
  const filtered = await getPage(scope, options, {
    page: 1,
    page_size: MAX_PAGE_SIZE,
    status,
    sort: "-createdAt",
  });
  assert.ok(
    filtered.total > 0,
    `${scope.key} real-status filter returned no items`,
  );
  assert.ok(
    filtered.items.every((item) => String(item.status) === status),
    `${scope.key} status filter returned a non-matching item`,
  );
  const filteredFacets = await getFacets(scope, options, { status });
  const statusFacet = filteredFacets.status.find(
    (item) => String(item.value) === status,
  );
  assert.equal(
    statusFacet?.count ?? 0,
    filtered.total,
    `${scope.key} status facet count differs from filtered total`,
  );

  return {
    total: descending.total,
    ids: new Set(allItems.map(itemID)),
    searchTotal: searched.total,
    status,
    statusTotal: filtered.total,
  };
}

function assertSubset(left, right, message) {
  for (const value of left)
    assert.ok(right.has(value), `${message}: missing ${value}`);
}

export async function runEMIASAcceptance({
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
    "unauthenticated EMIAS request must return HTTP 401",
  );

  const results = {};
  for (const scope of scopes) {
    const token = scope.tokenRole === "admin" ? adminToken : userToken;
    const result = await checkScope(scope, { baseURL, token, timeoutMs });
    results[scope.key] = result;
    log(
      `[PASS] ${scope.label}: total=${result.total}, search=${result.searchTotal}, ` +
        `status=${result.status}:${result.statusTotal}`,
    );
  }

  assertSubset(
    results.self.ids,
    results.account.ids,
    "self scope is not contained in account scope",
  );
  assertSubset(
    results.account.ids,
    results.admin.ids,
    "account scope is not contained in admin scope",
  );

  const summary = Object.fromEntries(
    Object.entries(results).map(([key, result]) => [
      key,
      {
        total: result.total,
        searchTotal: result.searchTotal,
        status: result.status,
        statusTotal: result.statusTotal,
      },
    ]),
  );
  log("[PASS] unauthenticated access rejected and scope containment verified");
  return { ok: true, scopes: summary };
}

function printHelp() {
  console.log(`Usage: CRATER_AUTH_TOKEN=... node hack/check-emias-pagination.mjs [options]

Runs read-only GET checks against real EMIAS pagination endpoints. Tokens are read
from environment variables and are never printed.

Environment:
  CRATER_API_BASE_URL          API root (default: ${DEFAULT_BASE_URL})
  CRATER_AUTH_TOKEN            Token used for both user and admin scopes
  CRATER_USER_AUTH_TOKEN       Optional token for user/account scopes
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
  await runEMIASAcceptance(options);
}

const entrypoint =
  process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url;
if (entrypoint) {
  main().catch((error) => {
    console.error(`FAIL: ${error.message}`);
    process.exitCode = 1;
  });
}
