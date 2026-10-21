import type { AgentConfig } from "../config.js";

export async function requestJobDiagnosticSnapshot(
  config: AgentConfig,
  contextToken: string,
  jobName: string,
): Promise<unknown> {
  const response = await fetch(
    `${config.backendInternalURL}/api/agent/jobs/${encodeURIComponent(jobName)}/diagnostic`,
    {
      headers: {
        Authorization: `Bearer ${config.internalToken}`,
        "X-Crater-Agent-Context": contextToken,
      },
    },
  );
  if (!response.ok) {
    throw new Error(
      `job diagnostic request returned status ${String(response.status)}`,
    );
  }
  const payload = (await response.json()) as {
    code?: number;
    data?: unknown;
    msg?: string;
  };
  if (payload.code !== 0 || payload.data === undefined) {
    throw new Error(payload.msg ?? "job diagnostic request failed");
  }
  return payload.data;
}
