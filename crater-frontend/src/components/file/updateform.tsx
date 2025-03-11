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
  FormDescription,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { apiDatasetUpdate } from "@/services/api/dataset"; // 假设有更新接口
import FormLabelMust from "@/components/form/FormLabelMust";
import { Badge } from "@/components/ui/badge";
import { X, Plus } from "lucide-react";
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
    .min(1, { message: "名称不能为空" })
    .max(256, { message: "名称最多包含 256 个字符" }),
  describe: z.string(),
  url: z.string(),
  type: z.enum(["dataset", "model"]),
  tags: z.array(z.string()).default([]), // 确保默认值为空数组
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
  const [inputValue, setInputValue] = React.useState("");

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
        ...values,
        name: values.datasetName,
        type: type ?? "dataset",
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

            <FormField
              control={form.control}
              name="tags"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{typestring}标签</FormLabel>
                  <FormControl>
                    <div className="grid gap-2">
                      <div className="mb-2 flex flex-wrap gap-2">
                        {field.value?.map((tag) => (
                          <Badge
                            key={tag}
                            variant="secondary"
                            className="gap-1"
                          >
                            {tag}
                            <X
                              className="h-3 w-3 cursor-pointer"
                              onClick={() => {
                                const newTags = field.value?.filter(
                                  (t) => t !== tag,
                                );
                                field.onChange(newTags);
                              }}
                            />
                          </Badge>
                        ))}
                      </div>
                      <div className="flex gap-2">
                        <Input
                          id="tags"
                          value={inputValue}
                          onChange={(e) => setInputValue(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              e.preventDefault();
                              if (inputValue.trim()) {
                                const newTags = [
                                  ...(field.value || []),
                                  inputValue.trim(),
                                ];
                                field.onChange(newTags);
                                setInputValue("");
                              }
                            }
                          }}
                          className="grow"
                        />
                        <Button
                          type="button"
                          onClick={() => {
                            if (inputValue.trim()) {
                              const newTags = [
                                ...(field.value || []),
                                inputValue.trim(),
                              ];
                              field.onChange(newTags);
                              setInputValue("");
                            }
                          }}
                          className="h-10 w-10 shrink-0 items-center justify-center rounded-full border border-gray-300 bg-gray-200 transition-colors duration-200 hover:bg-gray-300"
                        >
                          <Plus className="h-5 w-5 text-gray-600" />
                        </Button>
                      </div>
                    </div>
                  </FormControl>
                  <FormDescription>请输入标签</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
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
