import { IQuota } from "@/services/api/account";
import { z } from "zod";
import {
  convertKResourceToResource,
  convertResourceToKResource,
} from "./resource";

const nonNegativeNumberSchema = z
  .union([
    z.number().min(0, {
      message: "资源配额至少为 0",
    }),
    z.nan(),
  ])
  .optional();

export const quotaSchema = z.array(
  z.object({
    name: z.string().min(1, {
      message: "资源名称不能为空",
    }),
    guaranteed: nonNegativeNumberSchema,
    deserved: nonNegativeNumberSchema,
    capability: nonNegativeNumberSchema,
  }),
);

export type QuotaSchema = z.infer<typeof quotaSchema>;

export const convertQuotaToForm = (
  quota: IQuota,
  resourceTypes?: string[],
): QuotaSchema => {
  return (
    resourceTypes?.map((name) => ({
      name,
      guaranteed: convertKResourceToResource(name, quota.guaranteed?.[name]),
      deserved: convertKResourceToResource(name, quota.deserved?.[name]),
      capability: convertKResourceToResource(name, quota.capability?.[name]),
    })) ?? []
  );
};

export const convertFormToQuota = (form: QuotaSchema): IQuota => {
  const quota: IQuota = {
    guaranteed: {},
    deserved: {},
    capability: {},
  };

  form.forEach((resource) => {
    if (
      resource.guaranteed !== undefined &&
      resource.guaranteed !== null &&
      !isNaN(resource.guaranteed)
    ) {
      quota.guaranteed![resource.name] = convertResourceToKResource(
        resource.name,
        resource.guaranteed,
      );
    }
    if (
      resource.deserved !== undefined &&
      resource.deserved !== null &&
      !isNaN(resource.deserved)
    ) {
      quota.deserved![resource.name] = convertResourceToKResource(
        resource.name,
        resource.deserved,
      );
    }
    if (
      resource.capability !== undefined &&
      resource.capability !== null &&
      !isNaN(resource.capability)
    ) {
      quota.capability![resource.name] = convertResourceToKResource(
        resource.name,
        resource.capability,
      );
    }
  });

  return quota;
};
