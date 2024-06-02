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
  const setUserState = useSetAtom(globalUserInfo);
  const setAccount = useSetAtom(globalAccount);
  const { resetAll } = useResetStore();

  const lastView = useAtomValue(globalLastView);

  const { mutate: loginUser, status } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiUserLogin({
        username: values.username,
        password: values.password,
        auth: "act",
      }),
    onSuccess: async (data, { username }) => {
      await queryClient.invalidateQueries();
      setUserState({
        name: username,
      });
      setAccount(data.context);
      toast.success(
        `你好，${data.context.rolePlatform ? "管理员" : "用户"} ${username}`,
      );
      // navigate to /portal and clear all history
      const dashboard =
        lastView === "admin" && data.context.rolePlatform === Role.Admin
          ? "/admin"
          : lastView === "recommend"
            ? "/recommend"
            : "/portal";
      navigate(dashboard, { replace: true });
    },
  });

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: "",
      password: "",
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (status !== "pending") {
      resetAll();
      loginUser(values);
    }
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
        <LoadableButton
          type="submit"
          className="w-full"
          isLoading={status === "pending"}
        >
          登录
        </LoadableButton>
      </form>
    </Form>
  );
}
