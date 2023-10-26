// https://github.com/shadcn-ui/ui/issues/709

import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";

const formSchema = z.object({
  username: z
    .string()
    .min(1, {
      message: "Username can not be empty.",
    })
    .max(20, {
      message: "Username must be at most 20 characters.",
    }),
  password: z
    .string()
    .min(1, {
      message: "Password can not be empty.",
    })
    .max(20, {
      message: "Password must be at most 20 characters.",
    }),
});

interface TaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  closeSheet: () => void;
}

export function NewTaskForm({ closeSheet }: TaskFormProps) {
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
      username: "admin",
      password: "123456",
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    alert({ title: values.username });
    closeSheet();
  };

  return (
    <Form {...form}>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-2">
        <FormField
          control={form.control}
          name="username"
          render={({ field }) => (
            <FormItem className="flex flex-col">
              {/* <FormLabel>用户名</FormLabel> */}
              <FormControl>
                <Input
                  placeholder="用户名"
                  autoComplete="username"
                  {...field}
                />
                {/* <Input placeholder="shadcn" {...field} /> */}
              </FormControl>
              {/* <FormDescription>密码</FormDescription> */}
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="password"
          render={({ field }) => (
            <FormItem>
              {/* <FormLabel>Password</FormLabel> */}
              <FormControl>
                <Input
                  type="password"
                  placeholder="密码"
                  autoComplete="current-password"
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit">Save changes</Button>
      </form>
    </Form>
  );
}
