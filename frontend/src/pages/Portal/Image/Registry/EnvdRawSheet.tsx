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
import { z } from "zod";
import { toast } from "sonner";
import SandwichSheet, {
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/custom/LoadableButton";
import { PackagePlusIcon } from "lucide-react";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormDockerfile } from "@/components/form/types";
import { Input } from "@/components/ui/input";
import {
  apiUserCreateByEnvd,
  imageNameRegex,
  imageTagRegex,
} from "@/services/api/imagepack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import FormLabelMust from "@/components/form/FormLabelMust";
import { ImageSettingsFormCard } from "@/components/form/ImageSettingsFormCard";
import { DockerfileEditor } from "./DockerfileEditor";

export const envdRawFormSchema = z.object({
  envdScript: z.string().min(1, "Envd script content is required"),
  description: z.string().min(1, "请为镜像添加描述"),
  imageName: z
    .string()
    .min(1, "镜像名不能为空")
    .refine(
      (v) => {
        if (v) {
          return imageNameRegex.test(v);
        } else {
          return true;
        }
      },
      {
        message: "仅允许小写字母、数字、. _ -，且不能以分隔符开头/结尾",
      },
    )
    .refine((v) => !v || v.includes("envd"), {
      message: "名称需包含 'envd' ",
    }),
  imageTag: z
    .string()
    .min(1, "镜像标签不能为空")
    .refine(
      (v) => {
        if (v) {
          return imageTagRegex.test(v);
        } else {
          return true;
        }
      },
      {
        message: "仅允许字母、数字、_ . + -，且不能以 . 或 - 开头/结尾",
      },
    ),
});

export type EnvdRawFormValues = z.infer<typeof envdRawFormSchema>;

interface EnvdRawSheetContentProps {
  form: UseFormReturn<EnvdRawFormValues>;
  onSubmit: (values: EnvdRawFormValues) => void;
}

function EnvdRawSheetContent({ form, onSubmit }: EnvdRawSheetContentProps) {
  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-6">
      <FormField
        control={form.control}
        name="description"
        render={({ field }) => (
          <FormItem>
            <FormLabel>
              描述
              <FormLabelMust />
            </FormLabel>
            <FormControl>
              <Input {...field} />
            </FormControl>
            <FormDescription>
              关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示。
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <div className="grid grid-cols-2 gap-4">
        <FormField
          control={form.control}
          name="imageName"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                镜像名
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormDescription>自定义镜像名</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="imageTag"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                镜像标签
                <FormLabelMust />
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormDescription>自定义镜像标签</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
      <FormField
        control={form.control}
        name="envdScript"
        render={({ field }) => (
          <FormItem>
            <FormLabel>
              Envd 脚本
              <FormLabelMust />
            </FormLabel>
            <FormControl>
              <DockerfileEditor value={field.value} onChange={field.onChange} />
            </FormControl>
            <FormDescription>
              直接编写 envd 构建脚本，以 # syntax=v1 开头，包含 build() 函数
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
      <ImageSettingsFormCard
        form={form}
        imageNamePath="imageName"
        imageTagPath="imageTag"
        description="输入用户自定义的镜像名和镜像标签，若为空，则由系统自动生成"
      />
    </form>
  );
}

interface EnvdRawSheetProps extends SandwichSheetProps {
  closeSheet: () => void;
}

export function EnvdRawSheet({ closeSheet, ...props }: EnvdRawSheetProps) {
  const queryClient = useQueryClient();

  const form = useForm<EnvdRawFormValues>({
    resolver: zodResolver(envdRawFormSchema),
    defaultValues: {
      envdScript: `# syntax=v1

def build():
    base(image="crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.8.1-cudnn-devel-ubuntu22.04",dev=True)
    install.python(version="3.10")
    install.apt_packages(["openssh-server", "build-essential", "iputils-ping", "net-tools", "htop"])
    config.jupyter()`,
      description: "",
      imageName: "",
      imageTag: "",
    },
  });

  const { mutate: submitEnvdRawSheet, isPending } = useMutation({
    mutationFn: (values: EnvdRawFormValues) =>
      apiUserCreateByEnvd({
        description: values.description,
        envd: values.envdScript,
        name: values.imageName ?? "",
        tag: values.imageTag ?? "",
        python: "",
        base: "",
      }),
    onSuccess: async () => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ["imagepack", "list"] }),
      );
      closeSheet();
      toast.success(`镜像开始制作，请在下方列表中查看制作状态`);
    },
  });

  const onSubmit = (values: EnvdRawFormValues) => {
    submitEnvdRawSheet(values);
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
              isLoading={isPending}
              isLoadingText="正在提交"
              type="submit"
              onClick={async () => {
                const isValid = await form.trigger();
                if (isValid) {
                  form.handleSubmit(onSubmit)();
                }
              }}
            >
              <PackagePlusIcon />
              开始制作
            </LoadableButton>
          </>
        }
      >
        <EnvdRawSheetContent form={form} onSubmit={onSubmit} />
      </SandwichSheet>
    </Form>
  );
}
