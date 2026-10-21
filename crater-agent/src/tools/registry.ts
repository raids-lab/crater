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
        ? "After completing the user-facing answer, propose navigation to one allowlisted Crater administrator page as the final response item. Never call this before the answer and never use a /portal path."
        : "After completing the user-facing answer, propose navigation to one allowlisted Crater user page as the final response item. Never call this before the answer and never use an /admin path.",
    inputSchema: z.object({
      path: z.enum(paths),
      label: z.string().min(1).max(80),
      reason: z.string().min(1).max(300),
    }),
  });
}

const navigateToJobDetailTool = tool({
  description:
    "After completing the user-facing answer, propose opening the detail page for one exact Crater job as the final response item. Use the jobName returned by an authorized job query. The browser selects the administrator or user route from the current identity; never use navigate_to_page for a job detail.",
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
          ? "List authorized Crater jobs across the platform, newest first, with bounded pagination and optional owner, status, or text filters. Set pageSize to the user's requested limit; total is metadata and is not permission to fetch more."
          : "List the current user's Crater jobs in the active account, newest first, with bounded pagination and optional status or text filters. Set pageSize to the user's requested limit; total is metadata and is not permission to fetch more.",
      inputSchema: z.object({
        page: z.number().int().min(1).max(10_000).default(1),
        pageSize: z.number().int().min(1).max(50).default(20),
        search: z.string().trim().max(128).optional(),
        username: z.string().trim().min(1).max(64).optional(),
        statuses: z.array(z.enum(jobStatuses)).max(20).optional(),
      }),
      execute: async ({ page, pageSize, search, username, statuses }) => {
        const query = new URLSearchParams({
          page: String(page),
          pageSize: String(pageSize),
        });
        if (search) query.set("search", search);
        if (username) query.set("username", username);
        for (const status of statuses ?? []) query.append("status", status);
        return requestBackend(runtime, `/api/agent/jobs?${query.toString()}`);
      },
    }),
    get_job_diagnostic_snapshot: tool({
      description:
        runtime.identity === "admin"
          ? "Read a bounded diagnostic snapshot for one specified Crater job as an administrator. This is read-only and does not include raw logs. For a bounded list of jobs, use get_job_diagnostic_snapshots instead."
          : "Read a bounded diagnostic snapshot for one specified current-user Crater job. This never reads another user's job or raw logs. For a bounded list of jobs, use get_job_diagnostic_snapshots instead.",
      inputSchema: z.object({ jobName: z.string().trim().min(1).max(253) }),
      execute: async ({ jobName }) =>
        requestBackend(
          runtime,
          `/api/agent/jobs/${encodeURIComponent(jobName)}/diagnostic`,
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
      execute: async ({ jobNames }) => {
        const uniqueJobNames = [...new Set(jobNames)];
        const results: (
          | { jobName: string; status: "success"; data: unknown }
          | { jobName: string; status: "error"; error: string }
        )[] = [];
        for (const jobName of uniqueJobNames) {
          try {
            const data = await requestBackend(
              runtime,
              `/api/agent/jobs/${encodeURIComponent(jobName)}/diagnostic`,
            );
            results.push({ jobName, status: "success" as const, data });
          } catch (error) {
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
      execute: async ({ jobName }) =>
        requestBackend(
          runtime,
          `/api/agent/jobs/${encodeURIComponent(jobName)}/scheduling-diagnosis`,
        ),
    }),
    list_nodes: tool({
      description:
        "List every Crater cluster node visible to the current identity with sanitized scheduling capacity summaries.",
      inputSchema: z.object({}),
      execute: async () => requestBackend(runtime, "/api/agent/nodes"),
    }),
    get_node_capacity: tool({
      description:
        "Read one node's total, allocatable, requested, and estimated available CPU, memory, and accelerator resources.",
      inputSchema: z.object({ nodeName: z.string().trim().min(1).max(253) }),
      execute: async ({ nodeName }) =>
        requestBackend(
          runtime,
          `/api/agent/nodes/${encodeURIComponent(nodeName)}/capacity`,
        ),
    }),
    get_cluster_capacity: tool({
      description:
        "Read a point-in-time cluster-wide summary of node counts and allocatable, requested, and estimated available resources.",
      inputSchema: z.object({}),
      execute: async () => requestBackend(runtime, "/api/agent/capacity"),
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

async function requestBackend(runtime: ToolRuntime, path: string) {
  const response = await fetch(`${runtime.config.backendInternalURL}${path}`, {
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
  return payload.data;
}
