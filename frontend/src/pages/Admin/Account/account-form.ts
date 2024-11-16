import { IAccount } from "@/services/api/account";
import { convertQuotaToForm, quotaSchema } from "@/utils/quota";
import { z } from "zod";

export const formSchema = z.object({
  id: z.number().optional(),
  name: z
    .string()
    .min(1, {
      message: "账户名称不能为空",
    })
    .max(16, {
      message: "账户名称最多16个字符",
    }),
  resources: quotaSchema,
  expiredAt: z.date().optional(),
  admins: z.array(z.string()).optional(),
});

export type AccountFormSchema = z.infer<typeof formSchema>;

export const accountToForm = (
  resourceTypes: string[],
  account: IAccount,
): AccountFormSchema => {
  const formData: AccountFormSchema = {
    name: account.name,
    expiredAt: account.expiredAt ? new Date(account.expiredAt) : undefined,
    resources: convertQuotaToForm(account.quota, resourceTypes),
    admins: [],
  };

  return formData;
};
