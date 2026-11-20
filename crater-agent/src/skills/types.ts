export interface AgentSkill {
  readonly id: string;
  readonly instructions: string;
  readonly capabilities: readonly string[];
  readonly limitations: readonly string[];
  readonly activeTools: readonly string[];
}
