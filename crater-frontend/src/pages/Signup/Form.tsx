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
import { apiUserSignup } from "@/services/api/auth";
import LoadableButton from "@/components/custom/LoadableButton";
import { toast } from "sonner";

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
  const setUserState = useSetRecoilState(globalUserInfo);

  const { mutate: loginUser, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiUserSignup({
        userName: values.username,
        password: values.password,
        role: "user",
      }),
    onSuccess: async (data, { username }) => {
      await queryClient.invalidateQueries();
      const role = data.role === "admin" ? "admin" : "user";
      setUserState({
        id: username,
        role: role,
      });
      toast.success(
        `你好，${role === "admin" ? "管理员" : "用户"} ${username}`,
      );
      navigate("/portal");
    },
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
    !isPending && loginUser(values);
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
              <FormLabel>账号</FormLabel>
              <FormControl>
                <Input autoComplete="off" {...field} />
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
                <Input type="password" autoComplete="off" {...field} />
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
              <FormLabel>确认密码</FormLabel>
              <FormControl>
                <Input type="password" autoComplete="off" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <LoadableButton type="submit" className="w-full" isLoading={isPending}>
          注册
        </LoadableButton>
      </form>
    </Form>
  );
}
