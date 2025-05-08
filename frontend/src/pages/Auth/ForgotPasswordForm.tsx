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
import { useMutation } from "@tanstack/react-query";
import LoadableButton from "@/components/button/LoadableButton";
import { toast } from "sonner";
import { logger } from "@/utils/loglevel";

const formSchema = z.object({
  email: z
    .string()
    .min(1, {
      message: "邮箱不能为空",
    })
    .email({
      message: "请输入有效的邮箱地址",
    }),
});

export function ForgotPasswordForm() {
  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: "",
    },
  });

  const { mutate: sendResetEmail, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) => {
      // 这里添加发送重置密码邮件的API调用
      // 目前只是模拟API调用
      logger.info("Sending reset email", values);
      return new Promise<void>((resolve) => {
        setTimeout(() => resolve(), 1000);
      });
    },
    onSuccess: () => {
      toast.success("重置密码邮件已发送，请查收");
      form.reset();
    },
    onError: () => {
      toast.error("发送邮件失败，请稍后重试");
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    sendResetEmail(values);
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          control={form.control}
          name="email"
          render={({ field }) => (
            <FormItem>
              <FormLabel>邮箱地址</FormLabel>
              <FormControl>
                <Input type="email" autoComplete="email" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <LoadableButton
          isLoadingText="发送中"
          type="submit"
          className="w-full"
          isLoading={isPending}
        >
          发送重置链接
        </LoadableButton>
      </form>
    </Form>
  );
}
