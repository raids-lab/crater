import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { LabelInfo, apiNodeLabelsUpdate } from "@/services/api/nodelabel";
import * as React from "react";
import { Button } from "@/components/ui/button";
import FormLabelMust from "@/components/custom/FormLabelMust";

interface UpdateTaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
  current: LabelInfo;
}

const formSchema = z.object({
  id: z.number().int().positive(),
  name: z.string(),
  priority: z.number().int().positive(),
});

type FormSchema = z.infer<typeof formSchema>;

export function UpdateLabelForm({ closeSheet, current }: UpdateTaskFormProps) {
  const queryClient = useQueryClient();

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      id: current.id,
      name: current.name,
      priority: current.priority,
    },
  });

  const closeDialog = (event: { preventDefault: () => void }) => {
    event.preventDefault();
    closeSheet();
  };

  const { mutate: updateLabelPack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiNodeLabelsUpdate(values.id, values.name, values.priority),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["label", "list"],
      });
      toast.success(`Label ${name} 更新成功`);
      closeSheet();
    },
  });

  // 2. Define a submit handler.
  const onUpdateSubmit = (values: FormSchema) => {
    updateLabelPack(values);
  };

  return (
    <Form {...form}>
      <form
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onUpdateSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <div className="grid gap-3">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  别名
                  <FormLabelMust />
                </FormLabel>
                <Input {...field} />
                <FormDescription>
                  用于提交任务时，选择节点类型时使用
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="priority"
            render={() => (
              <FormItem>
                <FormLabel>
                  优先级
                  <FormLabelMust />
                </FormLabel>
                <Input
                  type="number"
                  {...form.register("priority", { valueAsNumber: true })}
                />
                <FormDescription>
                  优先级越大，意味着 GPU 性能越高
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="grid w-full grid-cols-2 gap-2">
            <Button onClick={closeDialog} variant={"secondary"}>
              取消
            </Button>
            <Button type="submit">提交</Button>
          </div>
        </div>
      </form>
    </Form>
  );
}
