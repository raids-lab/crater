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

export const dockerfileFormSchema = z.object({
  baseImage: z.string().min(1, "Base image is required"),
  requirements: z.string().min(1, "Requirements are required"),
  aptPackages: z.string().optional(),
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
        name="requirements"
        render={({ field }) => (
          <FormItem>
            <FormLabel>Python 依赖</FormLabel>
            <FormControl>
              <Textarea
                placeholder="粘贴 requirements.txt 文件内容到此处"
                className="h-40"
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
        name="aptPackages"
        render={({ field }) => (
          <FormItem>
            <FormLabel>APT Packages</FormLabel>
            <FormControl>
              <Textarea
                placeholder="e.g. git curl"
                className="h-40"
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
  const [isLoading] = useState(false);

  const form = useForm<DockerfileFormValues>({
    resolver: zodResolver(dockerfileFormSchema),
    defaultValues: {
      baseImage: "",
      requirements: "",
      aptPackages: "",
    },
  });

  const onSubmit = (values: DockerfileFormValues) => {
    // setIsLoading(true);
    toast.warning("正在生成 Dockerfile... " + values.toString());
    closeSheet();
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
              {isLoading ? "正在提交" : "开始制作"}
            </LoadableButton>
          </>
        }
      >
        <DockerfileSheetContent form={form} onSubmit={onSubmit} />
      </SandwichSheet>
    </Form>
  );
}
