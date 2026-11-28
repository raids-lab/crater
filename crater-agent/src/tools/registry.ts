import { tool, type ToolSet } from "ai";
import { z } from "zod";

import type { AgentConfig } from "../config.js";
import type { AgentIdentity } from "../page-context.js";

const userNavigationPaths = [
  "/portal/overview",
  "/portal/monitor/gpu",
  "/portal/monitor/idle",
  "/portal/monitor/network",
  "/portal/jobs/new/jupyter-job",
  "/portal/jobs/new/webide-job",
  "/portal/jobs/new/single-job",
  "/portal/jobs/new/pytorch-ddp-job",
  "/portal/jobs/new/tensorflow-ps-job",
  "/portal/jobs/inter",
  "/portal/jobs/custom",
  "/portal/templates",
  "/portal/env/images",
  "/portal/env/registry",
  "/portal/data/datasets",
  "/portal/data/models",
  "/portal/data/blocks",
  "/portal/files",
  "/portal/more/orders",
  "/portal/more/user",
] as const;

const adminNavigationPaths = [
  "/admin/jobs",
  "/admin/cluster/nodes",
  "/admin/cluster/resources",
  "/admin/monitor/gpu",
  "/admin/monitor/idle",
  "/admin/monitor/network",
  "/admin/env/registry",
  "/admin/data",
  "/admin/agent",
  "/admin/more",
] as const;

const jobStatuses = [
  "Prequeue",
  "Pending",
  "Aborting",
  "Aborted",
  "Running",
  "Restarting",
  "Completing",
  "Completed",
  "Terminating",
  "Terminated",
  "Failed",
  "Deleted",
  "Freed",
  "Cancelled",
] as const;

interface ToolRuntime {
  config: AgentConfig;
  contextToken: string;
  identity: AgentIdentity;
}

function createNavigationTool(identity: AgentIdentity) {
  const paths =
    identity === "admin" ? adminNavigationPaths : userNavigationPaths;
  return tool({
    description:
      identity === "admin"
        ? "After completing requested read queries and the user-facing answer, propose allowlisted Crater administrator destinations as final confirmation cards. For multiple requested pages, call this tool once per destination in the requested order; do not skip intermediate pages or claim only one card is supported. Each card needs user confirmation. Never use a /portal path."
        : "After completing requested read queries and the user-facing answer, propose allowlisted Crater user destinations as final confirmation cards. For multiple requested pages, call this tool once per destination in the requested order; do not skip intermediate pages or claim only one card is supported. Each card needs user confirmation. Never use an /admin path.",
    inputSchema: z.object({
      path: z.enum(paths),
      label: z.string().min(1).max(80),
      reason: z.string().min(1).max(300),
      assistForm: z
        .boolean()
        .default(false)
        .describe(
          "True only when the user requests form filling/configuration after navigation. False for navigation only, explanations, or an explicit request not to fill the form.",
        ),
    }),
  });
}

const navigateToJobDetailTool = tool({
  description:
    "After completing the user-facing answer, propose opening the detail page for one exact Crater job among the final confirmation cards. For multiple requested destinations, keep the requested order. Use the jobName returned by an authorized job query. The browser selects the administrator or user route from the current identity; never use navigate_to_page for a job detail.",
  inputSchema: z.object({
    jobName: z
      .string()
      .trim()
      .min(1)
      .max(253)
      .regex(/^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$/),
    label: z.string().min(1).max(80),
    reason: z.string().min(1).max(300),
  }),
});

const proposeFormPatchTool = tool({
  description:
    "After completing the user-facing answer, propose an allowlisted patch for the currently registered Crater form as the final response item. Never call this before the answer. The browser applies it only after user confirmation.",
  inputSchema: z.object({
    instanceId: z
      .uuid()
      .describe(
        "Copy the current form instanceId exactly. It prevents applying a proposal to a reopened form.",
      ),
    capabilityId: z.string().min(1).max(128),
    schemaVersion: z.number().int().positive(),
    summary: z.string().min(1).max(300),
    patches: z
      .array(
        z.object({
          path: z.string().min(1).max(128),
          value: z.unknown(),
          reason: z.string().min(1).max(300),
        }),
      )
      .min(1)
      .max(32),
  }),
});

function createReadToolRegistry(runtime: ToolRuntime): ToolSet {
  return {
    list_jobs: tool({
      description:
        runtime.identity === "admin"
          ? "List authorized Crater jobs across the platform, newest first, with bounded pagination and optional owner, status, or text filters. The username filter accepts an exact login username or a unique matching display/profile nickname (Chinese or English, case-insensitive for English nicknames); use the name the user actually supplied, never guess romanized usernames. A duplicate nickname requires the login username. Set pageSize to the user's requested limit. Only set includeTotal when the user asks for an exact historical count; omit it for ordinary lists."
          : "List the current user's Crater jobs in the active account, newest first, with bounded pagination and optional status or text filters. Set pageSize to the user's requested limit. Only set includeTotal when the user asks for an exact historical count; omit it for ordinary lists.",
      inputSchema: z.object({
        page: z.number().int().min(1).max(10_000).default(1),
        pageSize: z.number().int().min(1).max(50).default(20),
        includeTotal: z.boolean().default(false),
        search: z.string().trim().max(128).optional(),
        username: z.string().trim().min(1).max(64).optional(),
        statuses: z.array(z.enum(jobStatuses)).max(20).optional(),
        scope: z
          .enum(["active", "all"])
          .optional()
          .describe(
            "Use active for current/now jobs (all unfinished lifecycle phases). Use all for recent/history or explicit terminal statuses. Defaults to active unless statuses are supplied. A successful empty result is final; do not widen filters or query history to double-check it.",
          ),
      }),
      execute: async (
        { page, pageSize, includeTotal, search, username, statuses, scope },
        { abortSignal },
      ) => {
        const query = new URLSearchParams({
          page: String(page),
          pageSize: String(pageSize),
          scope: scope ?? (statuses?.length ? "all" : "active"),
        });
        if (includeTotal) query.set("includeTotal", "true");
        if (search) query.set("search", search);
        if (username) query.set("username", username);
        for (const status of statuses ?? []) query.append("status", status);
        return requestBackend(
          runtime,
          `/api/agent/jobs?${query.toString()}`,
          abortSignal,
        );
      },
    }),
    get_job_diagnostic_snapshot: tool({
      description:
        runtime.identity === "admin"
          ? "Read a bounded diagnostic snapshot for one specified Crater job as an administrator. This is read-only and does not include raw logs. For a bounded list of jobs, use get_job_diagnostic_snapshots instead."
          : "Read a bounded diagnostic snapshot for one specified current-user Crater job. This never reads another user's job or raw logs. For a bounded list of jobs, use get_job_diagnostic_snapshots instead.",
      inputSchema: z.object({ jobName: z.string().trim().min(1).max(253) }),
      execute: async ({ jobName }, { abortSignal }) =>
        requestBackend(
          runtime,
          `/api/agent/jobs/${encodeURIComponent(jobName)}/diagnostic`,
          abortSignal,
        ),
    }),
    get_job_diagnostic_snapshots: tool({
      description:
        runtime.identity === "admin"
          ? "Read diagnostic snapshots for a previously selected, bounded list of up to 10 authorized Crater jobs as an administrator. Use exactly the job names returned by the bounded list query and never expand the set from total counts."
          : "Read diagnostic snapshots for a previously selected, bounded list of up to 10 current-user Crater jobs. Use exactly the job names returned by the bounded list query and never expand the set from total counts.",
      inputSchema: z.object({
        jobNames: z.array(z.string().trim().min(1).max(253)).min(1).max(10),
      }),
      execute: async ({ jobNames }, { abortSignal }) => {
        const uniqueJobNames = [...new Set(jobNames)];
        const results: (
          | { jobName: string; status: "success"; data: unknown }
          | { jobName: string; status: "error"; error: string }
        )[] = [];
        for (const jobName of uniqueJobNames) {
          abortSignal?.throwIfAborted();
          try {
            const data = await requestBackend(
              runtime,
              `/api/agent/jobs/${encodeURIComponent(jobName)}/diagnostic`,
              abortSignal,
            );
            results.push({ jobName, status: "success" as const, data });
          } catch (error) {
            abortSignal?.throwIfAborted();
            results.push({
              jobName,
              status: "error" as const,
              error:
                error instanceof Error
                  ? error.message
                  : "job diagnostic request failed",
            });
          }
        }
        const failed = results.filter(
          (result) => result.status === "error",
        ).length;
        return {
          status: failed > 0 ? "partial-error" : "success",
          requested: uniqueJobNames.length,
          succeeded: results.filter((result) => result.status === "success")
            .length,
          failed,
          results,
        };
      },
    }),
    get_job_scheduling_diagnosis: tool({
      description:
        "Read the current scheduling-resource estimate for an authorized Pending Crater job. The result is a point-in-time estimate, not the scheduler's final decision.",
      inputSchema: z.object({ jobName: z.string().trim().min(1).max(253) }),
      execute: async ({ jobName }, { abortSignal }) =>
        requestBackend(
          runtime,
          `/api/agent/jobs/${encodeURIComponent(jobName)}/scheduling-diagnosis`,
          abortSignal,
        ),
    }),
    list_nodes: tool({
      description:
        "List Crater cluster nodes with Kubernetes-reported capacity and allocatable resources, Pod request totals, and allocatable-minus-requested estimates. These are not live usage, free disk bytes, GPU health, or a scheduling guarantee.",
      inputSchema: z.object({}),
      execute: async (_, { abortSignal }) =>
        requestBackend(runtime, "/api/agent/nodes", abortSignal),
    }),
    get_node_capacity: tool({
      description:
        "Read one node's Kubernetes Capacity, Allocatable, active Pod resource Requests, and max(Allocatable - Requests, 0) estimate. GPU counts are scheduling resource quantities, not proof of healthy/free cards; ephemeral-storage Allocatable is not current free disk space. Do not claim that an estimated GPU count is usable or that ephemeral-storage Allocatable is free disk.",
      inputSchema: z.object({ nodeName: z.string().trim().min(1).max(253) }),
      execute: async ({ nodeName }, { abortSignal }) =>
        requestBackend(
          runtime,
          `/api/agent/nodes/${encodeURIComponent(nodeName)}/capacity`,
          abortSignal,
        ),
    }),
    get_cluster_capacity: tool({
      description:
        "Read a point-in-time Kubernetes resource summary. Accelerator Capacity totals cover all nodes; Allocatable, Pod Requests and max(Allocatable - Requests, 0) cover only Ready, schedulable nodes. GPU estimates do not prove device health or free cards; ephemeral-storage figures are not current free disk bytes or a scheduling guarantee.",
      inputSchema: z.object({}),
      execute: async (_, { abortSignal }) =>
        requestBackend(runtime, "/api/agent/capacity", abortSignal),
    }),
  };
}

export function selectSkillTools(
  activeTools: readonly string[],
  config: AgentConfig,
  contextToken: string,
  identity: AgentIdentity,
): ToolSet {
  const runtime = { config, contextToken, identity };
  const registry = {
    navigate_to_page: createNavigationTool(identity),
    navigate_to_job_detail: navigateToJobDetailTool,
    propose_form_patch: proposeFormPatchTool,
    ...createReadToolRegistry(runtime),
  } as unknown as ToolSet;
  const selectedTools: ToolSet = {};
  for (const toolName of activeTools) {
    const selectedTool = registry[toolName];
    if (!selectedTool) {
      throw new Error(`Skill requested an unknown tool: ${toolName}`);
    }
    selectedTools[toolName] = selectedTool;
  }
  return selectedTools;
}

async function requestBackend(
  runtime: ToolRuntime,
  path: string,
  abortSignal?: AbortSignal,
) {
  const timeout = AbortSignal.timeout(15_000);
  const response = await fetch(`${runtime.config.backendInternalURL}${path}`, {
    signal: abortSignal ? AbortSignal.any([abortSignal, timeout]) : timeout,
    headers: {
      Authorization: `Bearer ${runtime.config.internalToken}`,
      "X-Crater-Agent-Context": runtime.contextToken,
    },
  });
  const payload = (await response.json().catch(() => undefined)) as
    | { code?: number; data?: unknown; msg?: string; error?: string }
    | undefined;
  if (!response.ok || payload?.code !== 0 || payload.data === undefined) {
    throw new Error(
      payload?.msg ??
        payload?.error ??
        `platform data request returned status ${String(response.status)}`,
    );
  }
  return boundToolResult(payload.data);
}

const maximumToolResultBytes = 48 * 1_024;

export function boundToolResult(data: unknown): unknown {
  if (
    Buffer.byteLength(JSON.stringify(data), "utf8") <= maximumToolResultBytes
  ) {
    return data;
  }
  // Keep complete JSON items and disclose omissions instead of clipping raw JSON.
  if (Array.isArray(data)) {
    const items: unknown[] = [];
    let bytes = 512;
    for (const item of data) {
      const size = Buffer.byteLength(JSON.stringify(item), "utf8") + 1;
      if (bytes + size > maximumToolResultBytes) break;
      items.push(item);
      bytes += size;
    }
    return {
      items,
      total: data.length,
      omitted: data.length - items.length,
      truncated: true,
      notice:
        "Partial result only. Do not infer that omitted records do not exist. Query a specific object for details.",
    };
  }
  return {
    truncated: true,
    notice:
      "Result exceeds the tool budget. Narrow the query or request fewer records; no complete result is available.",
  };
}
