import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import { useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiTrainingCreate } from "@/services/api/vcjob";
import { toast } from "sonner";
import { CirclePlus } from "lucide-react";
import FormLabelMust from "@/components/form/FormLabelMust";
import {
  volumeMountsSchema,
  envsSchema,
  taskSchema,
  convertToResourceList,
  nodeSelectorSchema,
  VolumeMountType,
  exportToJsonString,
  forwardsSchema,
  jobNameSchema,
} from "@/utils/form";
import { useState } from "react";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { Textarea } from "@/components/ui/textarea";
import { ImageFormField } from "@/components/form/ImageFormField";
import { VolumeMountsCard } from "@/components/form/DataMountFormField";
import { MetadataFormCustom } from "@/components/form/types";
import { ResourceFormFields } from "@/components/form/ResourceFormField";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import FormExportButton from "@/components/form/FormExportButton";
import { PublishConfigForm } from "./Publish";
import LoadableButton from "@/components/custom/LoadableButton";
import PageTitle from "@/components/layout/PageTitle";
import FormImportButton from "@/components/form/FormImportButton";
import { TemplateInfo } from "@/components/form/TemplateInfo";
import { OtherOptionsFormCard } from "@/components/form/OtherOptionsFormField";
import { EnvFormCard } from "@/components/form/EnvFormField";
import { ForwardFormCard } from "@/components/form/ForwardFormField";

const markdown = `## 运行规则

1. 如果申请了 GPU 资源，当过去 2 个小时 GPU 利用率为 0，我们将尝试发送告警信息给用户，建议用户检查作业是否正常运行。若此后半小时 GPU 利用率仍为 0，**系统将释放作业占用的资源**。
2. 当作业运行超过 4 天，我们将尝试发送告警信息给用户，提醒用户作业运行时间过长；若此后一天内用户未联系管理员说明情况并锁定作业，**系统将释放作业占用的资源**。

## 以普通用户运行

自定义作业默认以 \`root\` 用户运行，这种情况下，由于 Jupyter 交互式作业默认以普通用户运行，可能会导致权限问题。

为了帮助您以普通用户运行自定义作业，我们在容器内注入了 \`/crater-start.sh\` 脚本，您可以通过以下方式使用：

### 方法1：直接执行命令
在批处理命令前加上脚本调用：
\`\`\`bash
/crater-start.sh python your_script.py
\`\`\`

### 方法2：多步骤执行
执行脚本后，指定以普通用户权限运行命令，若直接执行命令，则仍以root权限执行：
\`\`\`bash
/crater-start.sh
su - \${NB_USER} -c 'your_command_1'
su - \${NB_USER} -c 'your_command_2'
...
\`\`\`

脚本内容如下：

\`\`\`bash
#!/bin/bash
set -euo pipefail  # Strict error handling: undefined vars and pipe errors

# ========== User & Group Configuration ==========
# Validate required environment variables
: "\${NB_USER:?NB_USER must be set}"
: "\${NB_UID:?NB_UID must be set}"
: "\${NB_GID:?NB_GID must be set}"

# Create group if not exists
if ! getent group "\${NB_GID}" &>/dev/null; then
    groupadd --force --gid "\${NB_GID}" --non-unique "\${NB_GROUP:-\${NB_USER}}"
fi

# Create user if not exists
if ! id "\${NB_USER}" &>/dev/null; then
    useradd --no-log-init \\
            --home "/home/\${NB_USER}" \\
            --shell /bin/zsh \\
            --uid "\${NB_UID}" \\
            --gid "\${NB_GID}" \\
            --groups 100 \\
            "\${NB_USER}"
else
    echo "User \${NB_USER} already exists, will not recreate"
fi

# ========== Directory & Permissions ==========
# Ensure proper home directory setup
mkdir -p "/home/\${NB_USER}"
chown "\${NB_UID}:\${NB_GID}" "/home/\${NB_USER}"
chmod 755 "/home/\${NB_USER}"  # Standard directory permissions

# Configure sudo access
SUDOERS_FILE="/etc/sudoers.d/nopasswd-\${NB_USER}"
if [ ! -f "\${SUDOERS_FILE}" ]; then
    echo "\${NB_USER} ALL=(ALL) NOPASSWD:ALL" > "\${SUDOERS_FILE}"
    chmod 440 "\${SUDOERS_FILE}"  # Secure sudoers file permissions
fi

# ========== Shell Environment ==========
# Initialize zsh configuration
ZSHRC_FILE="/home/\${NB_USER}/.zshrc"
if [ ! -f "\${ZSHRC_FILE}" ]; then
    touch "\${ZSHRC_FILE}"
    chown "\${NB_UID}:\${NB_GID}" "\${ZSHRC_FILE}"
fi

# ========== Command Execution ==========
# Execute with minimal preserved environment
if [ $# -gt 0 ]; then
    # If arguments passed, execute them as commands
    exec sudo --preserve-env --set-home --user "\${NB_USER}" \\
        LD_LIBRARY_PATH="\${LD_LIBRARY_PATH:-}" \\
        "$@"
else
    # Otherwise just switch user
    exec sudo --preserve-env --set-home --user "\${NB_USER}" \\
        LD_LIBRARY_PATH="\${LD_LIBRARY_PATH:-}" \\
        /bin/zsh
fi
\`\`\`
`;

const formSchema = z.object({
  jobName: jobNameSchema,
  task: taskSchema,
  envs: envsSchema,
  volumeMounts: volumeMountsSchema,
  nodeSelector: nodeSelectorSchema,
  alertEnabled: z.boolean().default(true),
  forwards: forwardsSchema,
});

type FormSchema = z.infer<typeof formSchema>;

const dataProcessor = (data: FormSchema) => {
  // Convert forwards to a format suitable for the API
  if (data.forwards === undefined || data.forwards === null) {
    data.forwards = [];
  }
  // if rdma is enabled, set it to false
  if (data.task.resource.network) {
    data.task.resource.network.enabled = false;
    data.task.resource.network.model = undefined;
  }
  if (!data.task.resource.network) {
    data.task.resource.network = {
      enabled: false,
      model: undefined,
    };
  }
  return data;
};

export const EnvCard = "环境变量";

export const Component = () => {
  const [envOpen, setEnvOpen] = useState<boolean>(false);
  const [otherOpen, setOtherOpen] = useState<boolean>(true);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAtomValue(globalUserInfo);
  const { mutate: createTask, isPending } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiTrainingCreate({
        name: values.jobName,
        resource: convertToResourceList(values.task.resource),
        image: values.task.image,
        shell: values.task.shell,
        command: values.task.command,
        workingDir: values.task.workingDir,
        volumeMounts: values.volumeMounts,
        envs: values.envs,
        forwards: values.forwards,
        alertEnabled: values.alertEnabled,
        selectors: values.nodeSelector.enable
          ? [
              {
                key: "kubernetes.io/hostname",
                operator: "In",
                values: [`${values.nodeSelector.nodeName}`],
              },
            ]
          : undefined,
        template: exportToJsonString(MetadataFormCustom, values),
      }),
    onSuccess: async (_, { jobName }) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["job"] }),
        queryClient.invalidateQueries({ queryKey: ["context", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
      toast.success(`作业 ${jobName} 创建成功`);
      navigate(-1);
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      jobName: "",
      task: {
        taskName: "training",
        replicas: 1,
        resource: {
          cpu: 1,
          gpu: {
            count: 0,
          },
          memory: 2,
        },
        image: "",
        shell: "bash",
        command: "",
        workingDir: "/home/" + user.name,
        ports: [],
      },
      volumeMounts: [
        {
          type: VolumeMountType.FileType,
          subPath: `user`,
          mountPath: `/home/${user.name}`,
        },
      ],
      envs: [],
      alertEnabled: true,
      nodeSelector: {
        enable: false,
      },
      forwards: [],
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (isPending) {
      toast.info("请勿重复提交");
      return;
    }
    createTask(values);
  };

  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="grid flex-1 items-start gap-4 md:gap-x-6 lg:grid-cols-3"
        >
          <PageTitle
            title="新建自定义作业"
            description="使用自定义作业进行训练、推理等任务"
            className="lg:col-span-3"
            tipContent={`版本 ${MetadataFormCustom.version}`}
          >
            <div className="items-centor flex w-fit flex-row justify-end gap-3">
              <FormImportButton
                metadata={MetadataFormCustom}
                form={form}
                dataProcessor={dataProcessor}
                afterImport={(data) => {
                  if (data.envs.length > 0) {
                    setEnvOpen(true);
                  }
                  if (data.nodeSelector.enable) {
                    setOtherOpen(true);
                  }
                }}
              />
              <FormExportButton metadata={MetadataFormCustom} form={form} />
              <PublishConfigForm
                config={MetadataFormCustom}
                configform={form}
              />
              <LoadableButton
                isLoading={isPending}
                isLoadingText="提交作业"
                type="submit"
              >
                <CirclePlus className="size-4" />
                提交作业
              </LoadableButton>
            </div>
          </PageTitle>
          <div className="flex flex-col gap-4 md:gap-6 lg:col-span-2">
            <Card>
              <CardHeader>
                <CardTitle>基本设置</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-5">
                <FormField
                  control={form.control}
                  name="jobName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        作业名称
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormDescription>
                        名称可重复，最多包含 40 个字符
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <ResourceFormFields
                  form={form}
                  cpuPath="task.resource.cpu"
                  memoryPath="task.resource.memory"
                  gpuCountPath="task.resource.gpu.count"
                  gpuModelPath="task.resource.gpu.model"
                />
                <ImageFormField form={form} name="task.image" />
                <FormField
                  control={form.control}
                  name="task.shell"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        Shell
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Select
                          onValueChange={field.onChange}
                          value={field.value}
                        >
                          <SelectTrigger className="w-1/4 font-mono">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="bash" className="font-mono">
                              bash
                            </SelectItem>
                            <SelectItem value="sh" className="font-mono">
                              sh
                            </SelectItem>
                          </SelectContent>
                        </Select>
                      </FormControl>
                      <FormDescription>
                        选择用于执行命令的
                        <span className="mx-0.5 font-mono">shell</span>，默认为
                        <span className="mx-0.5 font-mono">bash</span>
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="task.command"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>启动命令</FormLabel>
                      <FormControl>
                        <Textarea
                          {...field}
                          className="h-30 font-mono"
                          placeholder={`# Example
source conda.sh;
conda activate base;
./start.sh;`}
                        />
                      </FormControl>
                      <FormDescription>
                        可通过
                        <span className="mx-0.5 font-mono">;</span>{" "}
                        拆分多行命令。若不输入，将使用镜像的启动命令
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="task.workingDir"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        工作目录
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input {...field} className="font-mono" />
                      </FormControl>
                      <FormDescription>
                        默认为用户主目录
                        <span className="mx-0.5 font-mono">
                          /home/{user.name}
                        </span>
                        ，重启后数据不会丢失
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>
            <TemplateInfo
              form={form}
              metadata={MetadataFormCustom}
              uiStateUpdaters={[
                {
                  condition: (data) => data.envs.length > 0,
                  setter: setEnvOpen,
                  value: true,
                },
                {
                  condition: (data) =>
                    data.nodeSelector.enable || data.alertEnabled,
                  setter: setOtherOpen,
                  value: true,
                },
              ]}
              dataProcessor={dataProcessor}
              defaultMarkdown={markdown}
            />
          </div>
          <div className="flex flex-col gap-4 md:gap-6">
            <VolumeMountsCard form={form} />
            <ForwardFormCard form={form} />
            <EnvFormCard form={form} open={envOpen} setOpen={setEnvOpen} />
            <OtherOptionsFormCard
              form={form}
              alertEnabledPath="alertEnabled"
              nodeSelectorEnablePath="nodeSelector.enable"
              nodeSelectorNodeNamePath="nodeSelector.nodeName"
              open={otherOpen}
              setOpen={setOtherOpen}
            />
          </div>
        </form>
      </Form>
    </>
  );
};
