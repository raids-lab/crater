import { z } from "zod";

const environmentSchema = z.object({
  PORT: z.coerce.number().int().positive().max(65_535).default(8000),
  CRATER_BACKEND_INTERNAL_URL: z.url().default("http://localhost:8088"),
  CRATER_AGENT_INTERNAL_TOKEN: z.string().min(32),
});

export interface AgentConfig {
  port: number;
  backendInternalURL: string;
  internalToken: string;
}

export function loadConfig(
  environment: NodeJS.ProcessEnv = process.env,
): AgentConfig {
  const parsed = environmentSchema.parse(environment);
  return {
    port: parsed.PORT,
    backendInternalURL: parsed.CRATER_BACKEND_INTERNAL_URL.replace(/\/$/, ""),
    internalToken: parsed.CRATER_AGENT_INTERNAL_TOKEN,
  };
}
