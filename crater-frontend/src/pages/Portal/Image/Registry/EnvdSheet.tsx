import { useForm } from "react-hook-form";
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
import SandwichSheet, {
  SandwichLayout,
  SandwichSheetProps,
} from "@/components/sheet/SandwichSheet";
import LoadableButton from "@/components/button/LoadableButton";
import { PackagePlusIcon } from "lucide-react";
import FormImportButton from "@/components/form/FormImportButton";
import FormExportButton from "@/components/form/FormExportButton";
import { MetadataFormEnvdAdvanced } from "@/components/form/types";
import { Input } from "@/components/ui/input";
import {
  apiUserCreateByEnvd,
  ImageDefaultTags,
  imageNameRegex,
  ImagePackSource,
  imageTagRegex,
} from "@/services/api/imagepack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import FormLabelMust from "@/components/form/FormLabelMust";
import Combobox from "@/components/form/Combobox";
import { Textarea } from "@/components/ui/textarea";
import { ImageSettingsFormCard } from "@/components/form/ImageSettingsFormCard";
import { TagsInput } from "@/components/form/TagsInput";
import { exportToJsonString } from "@/utils/form";
import { toast } from "sonner";
import { useImageTemplateLoader } from "@/hooks/useTemplateLoader";

const envdFormSchema = z.object({
  python: z.string().min(1, "Python version is required"),
  base: z.string().min(1, "CUDA version is required"),
  description: z.string().min(1, "请为镜像添加描述"),
  aptPackages: z.string().optional(),
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
  imageName: z
    .string()
    .optional()
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
    .optional()
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
  tags: z
    .array(
      z.object({
        value: z.string(),
      }),
    )
    .optional(),
});

export type EnvdFormValues = z.infer<typeof envdFormSchema>;

interface EnvdSheetContentProps {
  closeSheet: () => void;
  imagePackName: string;
  setImagePackName: (imagePackName: string) => void;
}

function EnvdSheetContent({
  closeSheet,
  imagePackName,
  setImagePackName,
}: EnvdSheetContentProps) {
  const queryClient = useQueryClient();

  const form = useForm<EnvdFormValues>({
    resolver: zodResolver(envdFormSchema),
    defaultValues: {
      description: "",
      imageName: "",
      imageTag: "",
      tags: [],
    },
  });

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: EnvdFormValues) =>
      apiUserCreateByEnvd({
        description: values.description,
        envd: generateBuildScript(
          values.base,
          values.python,
          values.aptPackages
            ?.split(" ")
            .map((item) => item.trim())
            .filter(Boolean),
          values.requirements
            ?.split("\n")
            .map((item) => item.trim())
            .filter(Boolean),
        ),
        name: values.imageName ?? "",
        tag: values.imageTag ?? "",
        python: values.python,
        base:
          CUDA_BASE_IMAGE.find((image) => image.value === values.base)
            ?.imageLabel ?? "",
        tags: values.tags?.map((item) => item.value) ?? [],
        template: exportToJsonString(MetadataFormEnvdAdvanced, values),
        buildSource: ImagePackSource.EnvdAdvanced,
      }),
    onSuccess: async () => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ["imagepack", "list"] }),
      );
      closeSheet();
      toast.success(`镜像开始制作，请在下方列表中查看制作状态`);
    },
  });

  const onSubmit = (values: EnvdFormValues) => {
    submitDockerfileSheet(values);
  };

  useImageTemplateLoader({
    form: form,
    metadata: MetadataFormEnvdAdvanced,
    imagePackName: imagePackName,
    setImagePackName: setImagePackName,
  });

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <>
              <FormImportButton
                metadata={MetadataFormEnvdAdvanced}
                form={form}
              />
              <FormExportButton
                metadata={MetadataFormEnvdAdvanced}
                form={form}
              />
              <LoadableButton
                isLoading={isPending}
                isLoadingText="正在提交"
                type="submit"
              >
                <PackagePlusIcon />
                开始制作
              </LoadableButton>
            </>
          }
        >
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
                  关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <div className="grid grid-cols-2 gap-4">
            <FormField
              control={form.control}
              name="python"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    Python 版本
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Combobox
                      items={PYTHON_VERSIONS.map((version) => ({
                        label: version,
                        value: version,
                      }))}
                      current={field.value}
                      handleSelect={(value) => field.onChange(value)}
                      formTitle="Python版本"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="base"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    CUDA 版本
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Combobox
                      items={CUDA_BASE_IMAGE}
                      current={field.value}
                      handleSelect={(value) => field.onChange(value)}
                      formTitle="CUDA版本"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <TagsInput
            form={form}
            tagsPath="tags"
            label={`镜像标签`}
            description={`为镜像添加标签，以便分类和搜索`}
            customTags={ImageDefaultTags}
          />
          <FormField
            control={form.control}
            name="aptPackages"
            render={({ field }) => (
              <FormItem>
                <FormLabel>APT Packages</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder="git curl"
                    className="h-24 font-mono"
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  输入要安装的 APT 包，使用空格分隔多个包
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
                    placeholder={`transformers>=4.46.3
diffusers==0.31.0`}
                    className="h-24 font-mono"
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  请粘贴 requirements.txt 文件的内容，以便安装所需的 Python 包
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <ImageSettingsFormCard
            form={form}
            imageNamePath="imageName"
            imageTagPath="imageTag"
          />
        </SandwichLayout>
      </form>
    </Form>
  );
}

interface EnvdSheetProps extends SandwichSheetProps {
  closeSheet: () => void;
  imagePackName?: string;
  setImagePackName: (imagePackName: string) => void;
}

export function EnvdSheet({
  closeSheet,
  imagePackName = "",
  setImagePackName,
  ...props
}: EnvdSheetProps) {
  return (
    <SandwichSheet {...props}>
      <EnvdSheetContent
        closeSheet={closeSheet}
        imagePackName={imagePackName}
        setImagePackName={setImagePackName}
      />
    </SandwichSheet>
  );
}

const CUDA_BASE_IMAGE: {
  imageLabel: string;
  label: string;
  value: string;
}[] = [
  {
    imageLabel: "cu12.8.1",
    label: "CUDA 12.8.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.8.1-cudnn-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu12.6.3",
    label: "CUDA 12.6.3",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.6.3-cudnn-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu12.5.1",
    label: "CUDA 12.5.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.5.1-cudnn-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu12.4.1",
    label: "CUDA 12.4.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.4.1-cudnn-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu12.3.2",
    label: "CUDA 12.3.2",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.3.2-cudnn9-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu12.2.2",
    label: "CUDA 12.2.2",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.2.2-cudnn8-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu12.1.1",
    label: "CUDA 12.1.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.1.1-cudnn8-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu11.8.0",
    label: "CUDA 11.8.0",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.8.0-cudnn8-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu11.7.1",
    label: "CUDA 11.7.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.7.1-cudnn8-devel-ubuntu22.04",
  },
  {
    imageLabel: "cu11.6.1",
    label: "CUDA 11.6.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.6.1-cudnn8-devel-ubuntu20.04",
  },
  {
    imageLabel: "cu11.5.2",
    label: "CUDA 11.5.2",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.5.2-cudnn8-devel-ubuntu20.04",
  },
  {
    imageLabel: "cu11.4.3",
    label: "CUDA 11.4.3",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.4.3-cudnn8-devel-ubuntu20.04",
  },
  {
    imageLabel: "cu11.3.1",
    label: "CUDA 11.3.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.3.1-cudnn8-devel-ubuntu20.04",
  },
  {
    imageLabel: "cu11.2.2",
    label: "CUDA 11.2.2",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.2.2-cudnn8-devel-ubuntu20.04",
  },
  {
    imageLabel: "cu11.1.1",
    label: "CUDA 11.1.1",
    value:
      "crater-harbor.act.buaa.edu.cn/nvidia/cuda:11.1.1-cudnn8-devel-ubuntu20.04",
  },
  {
    imageLabel: "ubuntu22.04",
    label: "Ubuntu 22.04 (no CUDA)",
    value: "crater-harbor.act.buaa.edu.cn/library/ubuntu:22.04",
  },
  {
    imageLabel: "ubuntu20.04",
    label: "Ubuntu 20.04 (no CUDA)",
    value: "crater-harbor.act.buaa.edu.cn/library/ubuntu:20.04",
  },
  {
    imageLabel: "ubuntu18.04",
    label: "Ubuntu 18.04 (no CUDA)",
    value: "crater-harbor.act.buaa.edu.cn/library/ubuntu:18.04",
  },
];

const PYTHON_VERSIONS = [
  "3.13",
  "3.12",
  "3.11",
  "3.10",
  "3.9",
  "3.8",
  "3.7",
  "3.6",
];

const generateBuildScript = (
  baseImage: string,
  pythonVersion: string = "3.12",
  extraAptPackages: string[] = [],
  extraPythonPackages: string[] = [],
) => `# syntax=v1

def build():
    base(image="${baseImage}",dev=True)
    install.python(version="${pythonVersion}")
    install.apt_packages([
        "openssh-server", "build-essential", "iputils-ping", "net-tools", "htop", "tree"
    ])
    install.apt_packages([${extraAptPackages.map((item) => `"${item}"`)}])
    config.repo(
        url="https://github.com/tensorchord/envd",
        description="envd quick start example",
    )
    config.pip_index(url = "https://pypi.tuna.tsinghua.edu.cn/simple")
    install.python_packages(name = [${extraPythonPackages.map((item) => `"${item}"`)}])

    run(commands=[
      "chsh -s /bin/zsh root;",
      "git clone --depth 1 https://mirrors.tuna.tsinghua.edu.cn/git/ohmyzsh.git;",
      "ZSH=\\"/usr/share/.oh-my-zsh\\" CHSH=\\"no\\" RUNZSH=\\"no\\" REMOTE=https://mirrors.tuna.tsinghua.edu.cn/git/ohmyzsh.git sh ./ohmyzsh/tools/install.sh;",
      "chmod a+rx /usr/share/.oh-my-zsh/oh-my-zsh.sh;",
      "rm -rf ./ohmyzsh;",
      "git clone --depth=1 https://gitee.com/mirrors/zsh-syntax-highlighting.git /usr/share/.oh-my-zsh/custom/plugins/zsh-syntax-highlighting;",
      "git clone --depth=1 https://gitee.com/mirrors/zsh-autosuggestions.git /usr/share/.oh-my-zsh/custom/plugins/zsh-autosuggestions;",
      "echo \\"export skip_global_compinit=1\\" >> /etc/zsh/zshenv;",
      "echo \\"export ZSH=\\\\\\"/usr/share/.oh-my-zsh\\\\\\"\\" >> /etc/zsh/zshrc;",
      "echo \\"plugins=(git extract sudo jsontools colored-man-pages zsh-autosuggestions zsh-syntax-highlighting)\\" >> /etc/zsh/zshrc;",
      "echo \\"ZSH_THEME=\\\\\\"robbyrussell\\\\\\"\\" >> /etc/zsh/zshrc;",
      "echo \\"export ZSH_COMPDUMP=\\\\$ZSH/cache/.zcompdump-\\\\$HOST\\" >> /etc/zsh/zshrc;",
      "mkdir -p /etc/jupyter;",
      "echo \\"c.ServerApp.terminado_settings = {\\\\\\"shell_command\\\\\\": [\\\\\\"/bin/zsh\\\\\\"]}\\" >> /etc/jupyter/jupyter_server_config.py;",
      "echo \\"source \\\\$ZSH/oh-my-zsh.sh\\" >> /etc/zsh/zshrc;",
      "echo \\"zstyle \\\\\\":omz:update\\\\\\" mode disabled\\" >> /etc/zsh/zshrc;",
    ])
    config.jupyter()`;
