import { z } from "zod";

export const agentIdentitySchema = z.enum(["user", "admin"]);
export type AgentIdentity = z.infer<typeof agentIdentitySchema>;

const fieldSchema = z.object({
  path: z.string().min(1).max(128),
  label: z.string().min(1).max(512),
  valueType: z.enum([
    "string",
    "number",
    "boolean",
    "string-array",
    "option-array",
    "volume-mount-array",
    "forward-array",
  ]),
  description: z.string().max(512).optional(),
});

const formOptionSchema = z.object({
  value: z.string().min(1).max(512),
  label: z.string().min(1).max(512),
  tags: z.array(z.string().max(64)).max(16).optional(),
});

export const pageContextSchema = z.object({
  route: z
    .string()
    .max(512)
    .refine(
      (route) => route.startsWith("/portal/") || route.startsWith("/admin/"),
      "unsupported Crater page route",
    ),
  action: z.enum([
    "choose-job-type",
    "choose-template",
    "assist-form",
    "find-platform-resource",
    "recreate-job",
    "diagnose-job",
  ]),
  title: z.string().min(1).max(512),
  prompt: z.string().min(1).max(512),
  form: z
    .object({
      instanceId: z.uuid(),
      capabilityId: z.string().min(1).max(128),
      schemaVersion: z.number().int().positive(),
      title: z.string().min(1).max(512),
      fields: z.array(fieldSchema).min(1).max(64),
      currentValues: z.record(z.string(), z.unknown()),
      options: z
        .record(
          z.string(),
          z.object({
            items: z.array(formOptionSchema).max(512),
            total: z.number().int().nonnegative(),
          }),
        )
        .optional(),
    })
    .optional(),
  resource: z
    .object({
      kind: z.literal("job"),
      jobName: z.string().min(1).max(512),
      status: z.string().max(512).optional(),
      jobType: z.string().max(512).optional(),
    })
    .optional(),
});

export type AgentPageContext = z.infer<typeof pageContextSchema>;
