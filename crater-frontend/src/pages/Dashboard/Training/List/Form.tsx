// https://github.com/shadcn-ui/ui/issues/709

import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import { Button } from "@/components/ui/button";
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
import { useToast } from "@/components/ui/use-toast";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const formSchema = z.object({
  taskname: z
    .string()
    .min(1, {
      message: "任务名称不能为空",
    })
    .max(40, {
      message: "任务名称最多包含40个字符",
    }),
  cpu: z.number().int().positive({ message: "CPU 核心数至少为 1" }),
  gpu: z.number().int().min(0),
  memory: z.number().int().positive(),
  image: z.string().url(),
  dir: z.string(),
  shareDir: z.string(),
  command: z.string(),
  args: z.string(),
  priority: z.enum(["low", "high"], {
    invalid_type_error: "Select a priority",
    required_error: "Please select a priority",
  }),
});

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function NewTaskForm({ closeSheet }: TaskFormProps) {
  const { toast } = useToast();
  //   const { mutate: loginUser, isLoading } = useMutation({
  //     mutationFn: (values: z.infer<typeof formSchema>) =>
  //       loginUserFn({ username: values.username, password: values.password }),
  //     onSuccess: (_, { username }) => {
  //       setUserState((old) => {
  //         return {
  //           ...old,
  //           id: username,
  //           role: "user",
  //         };
  //       });
  //       toast({
  //         title: `登陆成功`,
  //         description: `你好，用户 ${username}`,
  //       });
  //     },
  //     onError: () => alert("login error"),
  //   });

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      taskname: "",
      cpu: 1,
      gpu: 0,
      memory: 8,
      image: "",
      dir: "",
      shareDir: "",
      command: "",
      args: "",
      priority: undefined,
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    toast({ title: values.taskname });
    closeSheet();
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
          name="taskname"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                任务名<span className="ml-1 text-red-500">*</span>
              </FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              {/* <FormMessage>请输入任务名称</FormMessage> */}
              <FormMessage />
            </FormItem>
          )}
        />
        <div>
          <div className="grid grid-cols-3 gap-4">
            <FormField
              control={form.control}
              name="cpu"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    CPU<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input type="number" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="gpu"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    GPU<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input type="number" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="memory"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    内存 (GB)<span className="ml-1 text-red-500">*</span>
                  </FormLabel>
                  <FormControl>
                    <Input type="number" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          {/* <FormDescription className="mt-2">
            请选择需要分配的机器配置
          </FormDescription> */}
        </div>
        <FormField
          control={form.control}
          name="image"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                镜像地址<span className="ml-1 text-red-500">*</span>
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
          name="command"
          render={({ field }) => (
            <FormItem>
              <FormLabel>执行命令</FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="args"
          render={({ field }) => (
            <FormItem>
              <FormLabel>命令参数</FormLabel>
              <FormControl>
                <Textarea {...field} className="resize-none" />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="priority"
          render={({ field }) => (
            <FormItem>
              <FormLabel>
                任务优先级<span className="ml-1 text-red-500">*</span>
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
                    <SelectItem value="low">低优先级</SelectItem>
                    <SelectItem value="high">高优先级</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <Button type="submit">提交任务</Button>
      </form>
    </Form>
  );
}
