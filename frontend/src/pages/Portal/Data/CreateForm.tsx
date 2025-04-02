import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
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
import { Badge } from "@/components/ui/badge";
import { X, Plus } from "lucide-react";
import * as React from "react";

import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { apiDatasetCreate } from "@/services/api/dataset";
import { Switch } from "@/components/ui/switch";
import { FileSelectDialog } from "@/components/file/FileSelectDialog";
import FormLabelMust from "@/components/form/FormLabelMust";

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
  tags: z.array(z.string()),
  weburl: z.string(),
  ispublic: z.boolean().default(true),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
  type?: "dataset" | "model";
}

export function DatasetCreateForm({ closeSheet, type }: TaskFormProps) {
  const queryClient = useQueryClient();
  const typestring = type === "model" ? "模型" : "数据集";
  const [inputValue, setInputValue] = React.useState("");
  const { mutate: createImagePack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiDatasetCreate({
        describe: values.describe,
        name: values.datasetName,
        url: values.url,
        type: type ?? "dataset",
        tags: values.tags,
        weburl: values.weburl,
        ispublic: values.ispublic,
      }),
    onSuccess: async (_, { datasetName }) => {
      await queryClient.invalidateQueries({
        queryKey: ["data", "mydataset"],
      });
      toast.success(`${typestring} ${datasetName} 创建成功`);
      closeSheet();
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      datasetName: "",
      url: "",
      describe: "",
      type: type ?? "dataset",
      tags: [],
      weburl: "",
      ispublic: true,
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    createImagePack(values);
  };

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className="flex flex-col space-y-4 px-6"
      >
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
                <Input {...field} />
              </FormControl>
              {/* <FormMessage>请输入仓库地址</FormMessage> */}
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
                <Input {...field} />
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
                  <FormDescription> 请选择文件或文件夹</FormDescription>
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
              <FormLabel>{typestring}标签 </FormLabel>
              <FormControl>
                <div className="grid gap-2">
                  <div className="mb-2 flex flex-wrap gap-2">
                    {field.value.map((tag) => (
                      <Badge key={tag} variant="secondary" className="gap-1">
                        {tag}
                        <X
                          className="h-3 w-3 cursor-pointer"
                          onClick={() => {
                            const newTags = field.value.filter(
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
                          const newTags = [...field.value, inputValue];
                          field.onChange(newTags);
                          setInputValue("");
                        }
                      }}
                      className="grow"
                    />
                    <Button
                      type="button"
                      onClick={() => {
                        const newTags = [...field.value, inputValue];
                        field.onChange(newTags);
                        setInputValue("");
                      }}
                      className="h-10 w-10 shrink-0 items-center justify-center rounded-full border border-gray-300 bg-gray-200 transition-colors duration-200 hover:bg-gray-300"
                    >
                      <Plus className="h-5 w-5 text-gray-600" />
                    </Button>
                  </div>
                </div>
              </FormControl>
              <FormDescription> 请输入标签</FormDescription>
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
                <Input {...field} />
              </FormControl>
              <FormDescription>
                {" "}
                请输入{typestring}开源仓库地址(如果有)
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="ispublic"
          render={({ field }) => (
            <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
              <FormLabel className="font-normal">公开{typestring}</FormLabel>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="grid">
          <Button type="submit">提交数据集</Button>
        </div>
      </form>
    </Form>
  );
}
