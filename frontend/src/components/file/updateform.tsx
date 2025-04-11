import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { FileSelectDialog } from "@/components/file/FileSelectDialog";
import { toast } from "sonner";
import { Pencil } from "lucide-react";
import { useEffect } from "react";
import * as React from "react";
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
import { apiDatasetUpdate } from "@/services/api/dataset"; // 假设有更新接口
import FormLabelMust from "@/components/form/FormLabelMust";
import { TagsInput } from "@/components/form/TagsInput";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

// 复用创建表单的Schema
const formSchema = z.object({
  datasetName: z
    .string()
    .min(1, {
      message: "名称不能为空",
    })
    .max(256, {
      message: "名称最多包含 256 个字符",
    }),
  describe: z.string(),
  url: z.string(),
  type: z.enum(["dataset", "model"]),
  tags: z
    .array(
      z.object({
        value: z.string(),
      }),
    )
    .optional(),
  weburl: z.string(),
  ispublic: z.boolean().default(true),
});

type FormSchema = z.infer<typeof formSchema>;

interface UpdateFormProps {
  type?: "dataset" | "model";
  initialData: FormSchema & { datasetId: number }; // 添加id字段
  onSuccess?: () => void; // 添加回调函数
}

export function DatasetUpdateForm({
  type,
  initialData,
  onSuccess,
}: UpdateFormProps) {
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = React.useState(false);
  const typestring = type === "model" ? "模型" : "数据集";

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      ...initialData,
      tags: initialData.tags || [], // 强制转换为数组
    },
  });

  useEffect(() => {
    const subscription = form.watch(() => {
      // console.log("Form values changed:", value);
    });
    return () => subscription.unsubscribe();
  }, [form]);

  const { mutate: updateDataset, isPending } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiDatasetUpdate({
        datasetId: initialData.datasetId, // 使用传入的ID
        describe: values.describe,
        name: values.datasetName,
        url: values.url,
        type: type ?? "dataset",
        tags: values.tags?.map((item) => item.value) ?? [],
        weburl: values.weburl,
        ispublic: values.ispublic,
      }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["data", "mydataset"] });
      toast.success(`${typestring} ${variables.datasetName} 更新成功`);
      setIsOpen(false);
      onSuccess?.();
    },
    onError: () => {
      toast.error("更新失败，请检查网络或数据格式");
    },
  });

  const onSubmit = async (values: FormSchema) => {
    try {
      await updateDataset(values);
    } catch {
      toast.error("更新失败，请检查网络或数据格式");
    }
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
        if (open) {
          // 只在对话框打开时重置表单，不要在关闭时重置
          form.reset({
            ...initialData,
            tags: initialData.tags || [],
          });
        }
      }}
    >
      <DialogTrigger asChild>
        <Button variant="outline" className="h-8 w-8" size="icon" title="更新">
          <Pencil size={16} strokeWidth={2} />
        </Button>
      </DialogTrigger>

      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>更新{typestring}</DialogTitle>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="datasetName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {typestring}名称
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      value={field.value || ""}
                      onChange={(e) => field.onChange(e.target.value)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="describe"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {typestring}描述
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      value={field.value || ""}
                      onChange={(e) => field.onChange(e.target.value)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div>
              <div className="grid grid-cols-2 gap-3">
                <FormField
                  control={form.control}
                  name="url"
                  render={({ field }) => (
                    <FormItem className="col-span-2">
                      <FormLabel>
                        {typestring}地址
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <FileSelectDialog
                          value={field.value.split("/").pop()}
                          handleSubmit={(item) => {
                            field.onChange(item.id);
                            form.setValue("url", item.id);
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>

            <TagsInput
              form={form}
              tagsPath="tags"
              label={`${typestring}标签`}
              description={`为${typestring}添加标签，以便分类和搜索`}
              customTags={[
                { value: "VLM" },
                { value: "LLAMA" },
                { value: "LLM" },
                { value: "QWEN" },
                { value: "DeepSeek" },
                { value: "机器学习" },
                { value: "深度学习" },
                { value: "数据科学" },
                { value: "自然语言处理" },
                { value: "计算机视觉" },
                { value: "强化学习" },
              ]}
            />

            <FormField
              control={form.control}
              name="weburl"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{typestring}仓库地址</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      value={field.value || ""}
                      onChange={(e) => field.onChange(e.target.value)}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="flex justify-end gap-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsOpen(false)}
              >
                取消
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? "更新中..." : `更新${typestring}`}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
