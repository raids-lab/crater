import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { useSetRecoilState } from "recoil";
import { userInfoState } from "@/utils/store";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiUserSignup } from "@/services/api/auth";
import { showErrorToast } from "@/utils/toast";
import LoadableButton from "@/components/LoadableButton";

const formSchema = z
  .object({
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
    passwordConfirm: z.string(),
  })
  .refine((data) => data.password === data.passwordConfirm, {
    message: "Passwords don't match",
    path: ["passwordConfirm"],
  });

export function SignupForm() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const setUserState = useSetRecoilState(userInfoState);

  const { mutate: loginUser, isLoading } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiUserSignup({
        userName: values.username,
        role: "admin",
        password: values.password,
      }),
    onSuccess: async (_, { username }) => {
      await queryClient.invalidateQueries();
      setUserState((old) => {
        return {
          ...old,
          id: username,
          role: "user",
        };
      });
      alert(username);
      navigate("/dashboard");
    },
    onError: (error) => showErrorToast("注册失败", error),
  });

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: "",
      password: "",
      passwordConfirm: "",
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
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-2">
        <FormField
          control={form.control}
          name="username"
          render={({ field }) => (
            <FormItem>
              {/* <FormLabel className=" text-left">Username</FormLabel> */}
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
        <FormField
          control={form.control}
          name="passwordConfirm"
          render={({ field }) => (
            <FormItem>
              {/* <FormLabel>Password</FormLabel> */}
              <FormControl>
                <Input
                  type="password"
                  placeholder="再次输入密码"
                  autoComplete="current-password"
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <LoadableButton type="submit" className="w-full" isLoading={isLoading}>
          注册
        </LoadableButton>
      </form>
    </Form>
  );
}
