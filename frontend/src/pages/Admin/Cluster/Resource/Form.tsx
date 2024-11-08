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
import {
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import * as React from "react";
import { Button } from "@/components/ui/button";
import FormLabelMust from "@/components/custom/FormLabelMust";
import { Resource, apiAdminResourceUpdate } from "@/services/api/resource";

interface UpdateTaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
  current: Resource;
}

const formSchema = z.object({
  id: z.number().int(),
  label: z.string().min(1, { message: "别名不能为空" }),
});

type FormSchema = z.infer<typeof formSchema>;

export function UpdateResourceForm({
  closeSheet,
  current,
}: UpdateTaskFormProps) {
  const queryClient = useQueryClient();

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      id: current.ID,
      label: current.label,
    },
  });

  const closeDialog = (event: { preventDefault: () => void }) => {
    event.preventDefault();
    closeSheet();
  };

  const { mutate: updateLabelPack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiAdminResourceUpdate(values.id, values.label),
    onSuccess: async (_, { label }) => {
      await queryClient.invalidateQueries({
        queryKey: ["resource", "list"],
      });
      toast.success(`Label ${label} 更新成功`);
      closeSheet();
    },
  });

  // 2. Define a submit handler.
  const onUpdateSubmit = (values: FormSchema) => {
    updateLabelPack(values);
  };

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle>编辑资源</DialogTitle>
        <DialogDescription>{current.label} 的详细信息</DialogDescription>
      </DialogHeader>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onUpdateSubmit)}
          className="grid gap-4"
        >
          <FormField
            control={form.control}
            name="label"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  别名
                  <FormLabelMust />
                </FormLabel>
                <Input {...field} />
                <FormDescription>用于提交作业时，选择节点类型</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <DialogFooter className="grid grid-cols-2">
            <Button onClick={closeDialog} variant={"secondary"}>
              取消
            </Button>
            <Button type="submit">提交</Button>
          </DialogFooter>
        </form>
      </Form>
    </DialogContent>
  );
}
