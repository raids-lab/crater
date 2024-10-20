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
import { useLocation, useNavigate } from "react-router-dom";
import { useAtomValue, useSetAtom } from "jotai";
import {
  globalLastView,
  globalAccount,
  globalUserInfo,
  useResetStore,
} from "@/utils/store";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Role, apiUserLogin } from "@/services/api/auth";
import LoadableButton from "@/components/custom/LoadableButton";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useEffect } from "react";

const formSchema = z.object({
  username: z
    .string()
    .min(1, {
      message: "用户名不能为空",
    })
    .max(20, {
      message: "用户名最多 20 个字符",
    })
    .refine(
      (value) => {
        const regex = /^[a-z0-9]+$/;
        return regex.test(value);
      },
      {
        message: "只能包含小写字母和数字",
      },
    ),
  password: z
    .string()
    .min(1, {
      message: "密码不能为空",
    })
    .max(20, {
      message: "密码最多 20 个字符",
    }),
});

export function LoginForm() {
  const location = useLocation();

  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const setUserState = useSetAtom(globalUserInfo);
  const setAccount = useSetAtom(globalAccount);
  const { resetAll } = useResetStore();

  const lastView = useAtomValue(globalLastView);

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: "",
      password: "",
    },
  });
  const currentValues = form.watch();

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (status !== "pending") {
      resetAll();
      loginUser({
        username: values.username,
        password: values.password,
        auth: "act-ldap",
      });
    }
  };

  const { mutate: loginUser, status } = useMutation({
    mutationFn: (values: {
      auth: string;
      username?: string;
      password?: string;
      token?: string;
    }) =>
      apiUserLogin({
        auth: values.auth,
        username: values.username,
        password: values.password,
        token: values.token,
      }),
    onSuccess: async (data) => {
      await queryClient.invalidateQueries();
      setUserState({
        ...data.user,
        space: data.context.space,
      });
      setAccount(data.context);
      toast.success(
        `你好，${data.context.rolePlatform ? "系统管理员" : "用户"}${data.user.nickname}`,
      );
      // navigate to /portal and clear all history
      const dashboard =
        lastView === "admin" && data.context.rolePlatform === Role.Admin
          ? "/admin"
          : "/portal/overview";
      navigate(dashboard, { replace: true });
    },
    onError: () => {
      form.setError("password", {
        type: "manual",
        message: "用户名或密码错误",
      });
    },
  });

  useEffect(() => {
    const searchParams = new URLSearchParams(location.search);
    const token = searchParams.get("token");
    if (token && token.length > 0) {
      loginUser({
        auth: "act-api",
        token,
      });
    }
  }, [location, loginUser]);

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
              <div className="flex items-center justify-between">
                <FormLabel>密码</FormLabel>
                <button
                  className="p-0 text-sm text-muted-foreground underline"
                  type="button"
                  onClick={() => toast.info("请联系 ACT 实验室账号管理员")}
                >
                  忘记密码？
                </button>
              </div>
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
        <LoadableButton
          type="submit"
          className="w-full"
          isLoading={status === "pending"}
        >
          ACT 登录
        </LoadableButton>
        <Button
          className="w-full"
          type="button"
          variant="outline"
          onClick={() => {
            if (status !== "pending") {
              resetAll();
              loginUser({
                username: currentValues.username,
                password: currentValues.password,
                auth: "normal",
              });
            }
          }}
        >
          其他登录方式
        </Button>
      </form>
    </Form>
  );
}
