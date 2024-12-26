import { useState } from "react";
import { useForm, UseFormReturn } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import { z } from "zod";
import { toast } from "sonner";
import SandwichSheet, {
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/custom/LoadableButton";
import { logger } from "@/utils/loglevel";
import { PackagePlusIcon } from "lucide-react";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormDockerfile } from "@/components/form/types";
import FormLabelMust from "@/components/form/FormLabelMust";
import { JobType } from "@/services/api/vcjob";
import Combobox from "@/components/form/Combobox";
import ImageItem from "@/components/form/ImageItem";
import useImageQuery from "@/hooks/query/useImageQuery";
import { Input } from "@/components/ui/input";
import { apiUserCreateKaniko } from "@/services/api/imagepack";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export const dockerfileFormSchema = z.object({
  baseImage: z.string().min(1, "基础镜像是必填项"),
  requirements: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          try {
            v.split("\n").forEach((line) => {
              if (line.trim().startsWith("#")) {
                return;
              }
              if (!line.trim()) {
                return;
              }
              // relation:
              // ==：等于
              // >：大于版本
              // >=：大于等于
              // <：小于版本
              // <=：小于等于版本
              // ~=：兼容版本

              // 基于 relation 将每一行的内容进行分割，分割后的内容为：name, relation, version
              // 可以只有 name
              const regex = /([a-zA-Z0-9_]+)([<>=!~]+)?([a-zA-Z0-9_.]+)?/;
              const match = line.match(regex);
              if (!match) {
                throw new Error("Invalid requirement format");
              }
              if (match.length < 2) {
                throw new Error("Invalid requirement format");
              }
            });
          } catch {
            return false;
          }
        }
        return true;
      },
      {
        message: "requirements.txt 文件格式无效",
      },
    ),
  aptPackages: z.string().optional(),
  description: z.string().optional(),
});

export type DockerfileFormValues = z.infer<typeof dockerfileFormSchema>;

export const baseImages = [
  {
    value: "nvidia/cuda:11.8.0-cudnn8-runtime-ubuntu22.04",
    label: "CUDA 11.8, Python 3.10, Pytorch 1.10",
  },
  {
    value: "nvidia/cuda:12.2.0-cudnn8-runtime-ubuntu22.04",
    label: "CUDA 12.2, Python 3.11, TensorFlow 2.8",
  },
  { value: "python:3.9-slim", label: "Python 3.9 (no CUDA)" },
  { value: "python:3.10-slim", label: "Python 3.10 (no CUDA)" },
];

interface DockerfileSheetContentProps {
  form: UseFormReturn<DockerfileFormValues>;
  onSubmit: (values: DockerfileFormValues) => void;
}

function DockerfileSheetContent({
  form,
  onSubmit,
}: DockerfileSheetContentProps) {
  const { data: images } = useImageQuery(JobType.Jupyter);

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-6">
      <FormField
        control={form.control}
        name="baseImage"
        render={({ field }) => (
          <FormItem>
            <FormLabel>
              基础镜像
              <FormLabelMust />
            </FormLabel>
            <FormControl autoFocus={true}>
              <Combobox
                items={images ?? []}
                current={field.value}
                handleSelect={(value) => field.onChange(value)}
                renderLabel={(item) => <ImageItem item={item} />}
                formTitle="镜像"
              />
            </FormControl>
            <FormDescription>
              选择一个带有所需 CUDA 和 Python 版本的基础镜像。
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="aptPackages"
        render={({ field }) => (
          <FormItem>
            <FormLabel>APT Packages</FormLabel>
            <FormControl>
              <Textarea
                placeholder="e.g. git curl"
                className="h-24"
                {...field}
              />
            </FormControl>
            <FormDescription>
              输入要安装的 APT 包，例如 git、curl 等。使用空格分隔多个包。
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="requirements"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Python 依赖</FormLabel>
            <FormControl>
              <Textarea
                placeholder="粘贴 requirements.txt 文件内容到此处"
                className="h-24"
                {...field}
              />
            </FormControl>
            <FormDescription>
              请粘贴 requirements.txt 文件的内容，以便安装所需的 Python
              包。点击帮助图标查看示例。
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <FormField
        control={form.control}
        name="description"
        render={({ field }) => (
          <FormItem>
            <FormLabel>备注</FormLabel>
            <FormControl>
              <Input {...field} />
            </FormControl>
            <FormDescription>
              关于此镜像的简短描述，如包含的软件版本、用途等。
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </form>
  );
}

interface DockerfileSheetProps extends SandwichSheetProps {
  closeSheet: () => void;
}

export function DockerfileSheet({
  closeSheet,
  ...props
}: DockerfileSheetProps) {
  const [isLoading, setIsLoading] = useState(false);
  const queryClient = useQueryClient();

  const form = useForm<DockerfileFormValues>({
    resolver: zodResolver(dockerfileFormSchema),
    defaultValues: {
      baseImage: "",
      requirements: "",
      aptPackages: "",
    },
  });

  const { mutate: submitDockerfileSheet } = useMutation({
    mutationFn: (values: DockerfileFormValues) =>
      apiUserCreateKaniko({
        description: values.description ? values.description : "",
        image: values.baseImage,
        requirements: values.requirements ? values.requirements : "",
        packages: values.aptPackages ? values.aptPackages : "",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["imagelink", "list"],
      });
      closeSheet();
      setIsLoading(false);
      toast.success(`镜像开始制作，请在下方列表中查看制作状态`);
    },
    onError: (error) => {
      setIsLoading(false);
      toast.error(`镜像制作请求提交失败：${error.message}`);
    },
  });

  const onSubmit = (values: DockerfileFormValues) => {
    setIsLoading(true);
    submitDockerfileSheet(values);
  };

  return (
    <Form {...form}>
      <SandwichSheet
        {...props}
        footer={
          <>
            <FormImportButton metadata={MetadataFormDockerfile} form={form} />
            <FormExportButton metadata={MetadataFormDockerfile} form={form} />
            <LoadableButton
              isLoading={isLoading}
              isLoadingText="正在提交"
              type="submit"
              onClick={() => {
                form
                  .trigger()
                  .then(() => {
                    if (form.formState.isValid) {
                      onSubmit(form.getValues());
                    }
                  })
                  .catch((e) => logger.debug(e));
              }}
            >
              <PackagePlusIcon />
              开始制作
            </LoadableButton>
          </>
        }
      >
        <DockerfileSheetContent form={form} onSubmit={onSubmit} />
      </SandwichSheet>
    </Form>
  );
}
