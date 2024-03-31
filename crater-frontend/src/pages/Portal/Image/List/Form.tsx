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

const formSchema = z.object({
  gitRepository: z.string().min(1, { message: "仓库地址不能为空" }),
  accessToken: z.string().min(1, { message: "AccessToken不能为空" }),
  registryServer: z.string(),
  registryUser: z.string(),
  registryPass: z.string(),
  registryProject: z.string(),
  imageName: z.string().min(1, { message: "镜像名不能为空" }),
  imageTag: z.string().min(1, { message: "标签不能为空" }),
  needProfile: z.boolean().default(false),
});

type FormSchema = z.infer<typeof formSchema>;

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function NewTaskForm({ closeSheet }: TaskFormProps) {
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
        <div className="grid grid-cols-3 gap-3">
          <div className="col-span-3">
            <FormField
              control={form.control}
              name="gitRepository"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像仓库<span className="ml-1 text-red-500">*</span>
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
                    AccessToken<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
        <div>
          <div className="grid grid-cols-3 gap-3">
            <FormField
              control={form.control}
              name="registryServer"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    仓库地址<span className="ml-1 text-red-500">*</span>
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
                    仓库账户<span className="ml-1 text-red-500">*</span>
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
              name="registryPass"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    仓库密码<span className="ml-1 text-red-500">*</span>
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
                <FormItem>
                  <FormLabel>
                    仓库项目<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
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
            name="imageName"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  镜像名<span className="ml-1 text-red-500">*</span>
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
                <FormLabel>镜像标签</FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="needProfile"
            render={({ field }) => (
              <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                <div className="space-y-0.5">
                  <FormLabel className="text-base">是否需要Profile</FormLabel>
                  {/* <FormDescription>
                    Receive emails about new products, features, and more.
                  </FormDescription> */}
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Button type="submit">提交镜像</Button>
        </div>
      </form>
    </Form>
  );
}
