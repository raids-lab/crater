import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import * as z from "zod";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Card,
  CardTitle,
  CardContent,
  CardFooter,
  CardHeader,
  CardDescription,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useState } from "react";
import { toast } from "sonner";
import { useAtom } from "jotai";
import { globalUserInfo } from "@/utils/store";
import { useMutation } from "@tanstack/react-query";
import { IUserAttributes } from "@/services/api/admin/user";
import {
  apiContextUpdateUserAttributes,
  apiSendVerificationEmail,
  apiVerifyEmailCode,
} from "@/services/api/context";
import { MailPlusIcon, UserRoundCogIcon } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSeparator,
  InputOTPSlot,
} from "@/components/ui/input-otp";
import LoadableButton from "@/components/custom/LoadableButton";

const formSchema = z.object({
  nickname: z.string().min(2, {
    message: "Nickname must be at least 2 characters.",
  }),
  email: z.string().email().optional().nullable(),
  teacher: z.string().optional().nullable(),
  group: z.string().optional().nullable(),
  expiredAt: z.string().optional().nullable(),
  phone: z.string().optional().nullable(),
  avatar: z.string().url().optional().nullable(),
});

export default function UserSettings() {
  const [user, setUser] = useAtom(globalUserInfo);
  const [avatarPreview, setAvatarPreview] = useState(user.avatar || "");
  const [originalEmail, setOriginalEmail] = useState(user.email || "");
  const [isEmailVerified, setIsEmailVerified] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isVerifyError, setIsVerifyError] = useState(false);
  const [verificationCode, setVerificationCode] = useState("");

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      nickname: user.nickname,
      email: user.email || null,
      teacher: user.teacher || null,
      group: user.group || null,
      expiredAt: user.expiredAt || null,
      phone: user.phone || null,
      avatar: user.avatar || null,
    },
  });

  const { mutate: updateUser } = useMutation({
    mutationFn: (values: IUserAttributes) =>
      apiContextUpdateUserAttributes(values),
    onSuccess: (_data, values) => {
      toast.success("用户信息已更新");
      setUser((prev) => ({ ...prev, ...values }));
    },
  });

  const {
    mutate: sendVerificationEmail,
    isPending: isSendVerificationPending,
  } = useMutation({
    mutationFn: (email: string) => apiSendVerificationEmail(email),
    onSuccess: () => {
      setIsVerifyError(false);
      setVerificationCode("");
      setIsDialogOpen(true);
    },
  });

  const { mutate: verifyEmailCode } = useMutation({
    mutationFn: ({ email, code }: { email: string; code: string }) =>
      apiVerifyEmailCode(email, code),
    onError: () => {
      setIsVerifyError(true);
    },
    onSuccess: () => {
      toast.success("新邮箱已验证");
      setOriginalEmail(form.getValues("email") ?? "");
      setIsEmailVerified(true);
      setIsDialogOpen(false);
    },
  });

  function onSubmit(values: z.infer<typeof formSchema>) {
    if (!isEmailVerified && values.email !== originalEmail) {
      form.setError("email", {
        type: "manual",
        message: "请先验证您的新邮箱地址",
      });
      return;
    }

    updateUser({
      id: user.id,
      name: user.name,
      email: values.email ?? user.email,
      nickname: values.nickname ?? user.nickname,
      teacher: values.teacher ?? user.teacher,
      group: values.group ?? user.group,
      expiredAt: values.expiredAt ?? user.expiredAt,
      phone: values.phone ?? user.phone,
      avatar: values.avatar ?? user.avatar,
    });
  }

  return (
    <>
      <Card className="lg:col-span-2">
        <CardHeader>
          <CardTitle>用户信息</CardTitle>
          <CardDescription>
            更新您的用户信息，以便我们更好地为您服务
          </CardDescription>
        </CardHeader>
        <Separator />
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <CardContent className="space-y-8 pt-6">
              <div className="flex flex-row items-center gap-6">
                <FormField
                  control={form.control}
                  name="avatar"
                  render={({ field }) => (
                    <FormItem className="flex-1">
                      <FormLabel>头像</FormLabel>
                      <FormControl>
                        <div className="flex items-center space-x-4">
                          <Input
                            {...field}
                            value={field.value || ""}
                            placeholder="Avatar URL"
                            className="font-mono"
                            onChange={(e) => {
                              field.onChange(e);
                              setAvatarPreview(e.target.value);
                            }}
                          />
                        </div>
                      </FormControl>
                      <FormDescription>
                        通过图床上传图片，然后将图片链接粘贴到此处
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Avatar className="h-20 w-20">
                  <AvatarImage src={avatarPreview} alt="Avatar preview" />
                  <AvatarFallback>{user.name.charAt(0)}</AvatarFallback>
                </Avatar>
              </div>
              <Separator />
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>邮箱</FormLabel>
                    <FormControl>
                      <div className="flex items-center space-x-2">
                        <Input
                          {...field}
                          type="email"
                          className="font-mono"
                          value={field.value || ""}
                          onChange={(e) => {
                            field.onChange(e);
                            setIsEmailVerified(false);
                          }}
                        />
                        {field.value !== originalEmail && (
                          <LoadableButton
                            isLoading={isSendVerificationPending}
                            isLoadingText="验证中"
                            variant="secondary"
                            type="button"
                            onClick={() => {
                              sendVerificationEmail(field.value || "");
                            }}
                          >
                            <MailPlusIcon />
                            验证
                          </LoadableButton>
                        )}
                      </div>
                    </FormControl>
                    <FormDescription>用于接收通知的邮箱地址</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
            <Separator />
            <CardFooter className="px-6 py-4">
              <Button type="submit">
                <UserRoundCogIcon />
                更新用户信息
              </Button>
            </CardFooter>
          </form>
        </Form>
        <AlertDialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>验证您的邮箱</AlertDialogTitle>

              {isVerifyError ? (
                <AlertDialogDescription className="text-destructive">
                  验证码错误，请重新检查
                </AlertDialogDescription>
              ) : (
                <AlertDialogDescription>
                  我们向<span>{form.getValues("email")}</span>
                  发送了一封验证邮件，请输入邮件中的验证码
                </AlertDialogDescription>
              )}
            </AlertDialogHeader>
            <div className="flex items-center justify-center">
              <InputOTP
                maxLength={6}
                value={verificationCode}
                onChange={(value) => setVerificationCode(value)}
              >
                <InputOTPGroup>
                  <InputOTPSlot index={0} aria-placeholder=" " />
                  <InputOTPSlot index={1} aria-placeholder=" " />
                  <InputOTPSlot index={2} aria-placeholder=" " />
                </InputOTPGroup>
                <InputOTPSeparator />
                <InputOTPGroup>
                  <InputOTPSlot index={3} aria-placeholder=" " />
                  <InputOTPSlot index={4} aria-placeholder=" " />
                  <InputOTPSlot index={5} aria-placeholder=" " />
                </InputOTPGroup>
              </InputOTP>
            </div>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <Button
                onClick={() =>
                  verifyEmailCode({
                    code: verificationCode,
                    email: form.getValues("email") ?? "",
                  })
                }
              >
                验证
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </Card>
    </>
  );
}
