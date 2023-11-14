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
import { useNavigate } from "react-router-dom";
import { useSetRecoilState } from "recoil";
import { globalUserInfo } from "@/utils/store";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiUserLogin } from "@/services/api/auth";
import { useToast } from "@/components/ui/use-toast";
import { showErrorToast } from "@/utils/toast";
import LoadableButton from "@/components/LoadableButton";

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

export function ProfileForm() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const setUserState = useSetRecoilState(globalUserInfo);
  const { toast } = useToast();

  const { mutate: loginUser, isLoading } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiUserLogin({ username: values.username, password: values.password }),
    onSuccess: async (data, { username }) => {
      await queryClient.invalidateQueries();
      const role = data.role === "admin" ? "admin" : "user";
      setUserState({
        id: username,
        role: role,
      });
      toast({
        title: `登陆成功`,
        description: `你好，${
          role === "admin" ? "管理员" : "用户"
        } ${username}`,
      });
      // navigate to /portal and clear all history
      navigate("/portal", { replace: true });
    },
    onError: (error) => showErrorToast("登陆失败", error),
  });

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: "test",
      password: "123456",
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    !isLoading && loginUser(values);
  };

  return (
    <Form {...form}>
      {/* eslint-disable-next-line @typescript-eslint/no-misused-promises */}
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          control={form.control}
          name="username"
          render={({ field }) => (
            <FormItem>
              <FormLabel>账号</FormLabel>
              <FormControl>
                <Input autoComplete="username" {...field} />
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
              <FormLabel>密码</FormLabel>
              <FormControl>
                <Input
                  type="password"
                  autoComplete="current-password"
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <LoadableButton type="submit" className="w-full" isLoading={isLoading}>
          登录
        </LoadableButton>
      </form>
    </Form>
  );
}
