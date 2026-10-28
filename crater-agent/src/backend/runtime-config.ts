import type { AgentConfig } from "../config.js";
import {
  agentRuntimeConfigSchema,
  defaultAgentRuntimeConfig,
  type AgentRuntimeConfig,
} from "../runtime-config.js";

export async function requestAgentRuntimeConfig(
  config: AgentConfig,
): Promise<AgentRuntimeConfig> {
  try {
    const response = await fetch(
      `${config.backendInternalURL}/api/agent/runtime-config`,
      {
        headers: { Authorization: `Bearer ${config.internalToken}` },
        signal: AbortSignal.timeout(3_000),
      },
    );
    if (!response.ok) {
      throw new Error(`runtime config returned ${String(response.status)}`);
    }
    const payload: unknown = await response.json();
    const parsed = agentRuntimeConfigSchema.safeParse(
      typeof payload === "object" && payload !== null && "data" in payload
        ? payload.data
        : undefined,
    );
    if (parsed.success) {
      return parsed.data;
    }
  } catch (error) {
    console.error("Crater-Agent runtime config fallback", error);
  }
  return defaultAgentRuntimeConfig;
}
