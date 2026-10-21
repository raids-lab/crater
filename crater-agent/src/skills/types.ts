export interface AgentSkill {
  readonly id: string;
  readonly instructions: string;
  readonly capabilities: readonly string[];
  readonly limitations: readonly string[];
  readonly activeTools: readonly string[];
  readonly staticResponse?: string;
}

export const platformReadTools = [
  "list_jobs",
  "get_job_diagnostic_snapshot",
  "get_job_diagnostic_snapshots",
  "get_job_scheduling_diagnosis",
  "list_nodes",
  "get_node_capacity",
  "get_cluster_capacity",
] as const;
