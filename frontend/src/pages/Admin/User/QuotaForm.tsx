import { useToast } from "@/components/ui/use-toast";
import { apiAdminUserUpdateQuota } from "@/services/api/admin/user";
import { getKubernetesResource, KResource } from "@/utils/resource";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

const formSchema = z.object({
  cpu: z.coerce.number().int().positive({ message: "CPU 核心数至少为 1" }),
  gpu: z.coerce.number().int().min(0),
  memory: z.coerce.number().int().positive(),
});

type FormSchema = z.infer<typeof formSchema>;

interface QuotaFormProps {
  closeSheet: () => void;
  userName: string;
  quota: KResource;
}

export const QuotaForm = ({ userName, closeSheet, quota }: QuotaFormProps) => {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const { mutate: updateQuota } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiAdminUserUpdateQuota({
        userName: userName,
        hardQuota: getKubernetesResource({
          gpu: values.gpu,
          memory: `${values.memory}Gi`,
          cpu: values.cpu,
        }),
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "userlist"] });
      toast({
        title: `更新成功`,
        description: `用户「${userName}」配额已更新`,
      });
      closeSheet();
    },
  });

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      cpu: quota.cpu ?? 0,
      gpu: quota.gpu ?? 0,
      memory: quota.memoryNum ?? 0,
    },
  });

  const onSubmit = (values: FormSchema) => {
    // if quota has not changed, do not submit
    if (
      values.cpu === quota.cpu &&
      values.gpu === quota.gpu &&
      values.memory === quota.memoryNum
    ) {
      closeSheet();
      return;
    }
    updateQuota(values);
  };

  return (
    <Form {...form}>
      <form
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <FormField
          control={form.control}
          name="cpu"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                CPU<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input type="number" {...field} defaultValue={0} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="gpu"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                GPU<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input type="number" {...field} defaultValue={0} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="memory"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                内存 (GB)<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input type="number" {...field} defaultValue={0} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit">提交修改</Button>
      </form>
    </Form>
  );
};
