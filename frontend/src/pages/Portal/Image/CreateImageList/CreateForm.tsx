import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { apiUserImagepackCreate } from "@/services/api/imagepack";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import FormLabelMust from "@/components/custom/FormLabelMust";

const formSchema = z.object({
  gitRepository: z.string().url({ message: "仓库地址应为合法 URL" }),
  accessToken: z.string().min(1, { message: "Access Token 不能为空" }),
  registryServer: z.string(),
  registryUser: z.string(),
  registryPass: z.string(),
  registryProject: z.string(),
  imageName: z.string().min(1, { message: "镜像名不能为空" }),
  imageTag: z.string().min(1, { message: "标签不能为空" }),
  alias: z.string().min(1, { message: "镜像别名不能为空" }),
  description: z.string().min(1, { message: "镜像描述不能为空" }),
  taskType: z.enum(["1", "2"]),
  needProfile: z.boolean().default(false),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function ImageCreateForm({ closeSheet }: TaskFormProps) {
  const queryClient = useQueryClient();

  const { mutate: createImagePack } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiUserImagepackCreate({
        gitRepository: values.gitRepository,
        accessToken: values.accessToken,
        registryServer: values.registryServer,
        registryUser: values.registryUser,
        registryPass: values.registryPass,
        registryProject: values.registryProject,
        imageName: values.imageName,
        imageTag: values.imageTag,
        needProfile: values.needProfile,
        alias: values.alias,
        description: values.description,
        taskType: Number(values.taskType),
      }),
    onSuccess: async (_, { imageName, imageTag }) => {
      await queryClient.invalidateQueries({
        queryKey: ["imagepack", "list"],
      });
      toast.success(`镜像 ${imageName}:${imageTag} 创建成功`);
      closeSheet();
    },
  });

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      gitRepository: "",
      accessToken: "",
      registryServer: "",
      registryUser: "",
      registryPass: "",
      registryProject: "",
      imageName: "",
      imageTag: "",
      needProfile: false,
      alias: "",
      description: "",
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
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        onSubmit={form.handleSubmit(onSubmit)}
        className="mt-6 flex flex-col space-y-4"
      >
        <FormField
          control={form.control}
          name="gitRepository"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                Git 仓库地址
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
          name="accessToken"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                Access Token
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
              name="registryServer"
              render={({ field }) => (
                <FormItem className="col-span-2">
                  <FormLabel>
                    镜像仓库地址
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="registryProject"
              render={({ field }) => (
                <FormItem className="col-span-2">
                  <FormLabel>
                    镜像仓库项目
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="registryUser"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像仓库账户
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="dockerUsername"
                      id="dockerUsername"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="registryPass"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像仓库密码
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input
                      id="dockerPassword"
                      type="password"
                      autoComplete="dockerPassword"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="imageName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像名称
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} className="font-mono" />
                  </FormControl>
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
                    <Input {...field} className="font-mono" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="alias"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像别名
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} className="font-mono" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像描述
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} className="font-mono" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
        <div>
          <FormField
            control={form.control}
            name="taskType"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  任务类型
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Select
                    onValueChange={field.onChange}
                    defaultValue={field.value}
                  >
                    <SelectTrigger className="">
                      <SelectValue placeholder="请选择" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">离线任务</SelectItem>
                      <SelectItem value="2">Jupyter交互式任务</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
        <div>
          <FormField
            control={form.control}
            name="needProfile"
            render={({ field }) => (
              <FormItem>
                <FormLabel>更多设置</FormLabel>
                <div className="flex w-full flex-row items-center justify-between rounded-md border p-4 shadow-sm">
                  <div className="text-sm">启用 Profile 功能</div>
                  <FormControl>
                    <Switch
                      name={field.name}
                      id={field.name}
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </div>
              </FormItem>
            )}
          />
        </div>
        <div className="grid">
          <Button type="submit">提交镜像</Button>
        </div>
      </form>
    </Form>
  );
}
