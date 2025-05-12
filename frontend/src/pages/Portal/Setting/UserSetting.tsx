// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
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
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useEffect, useState } from "react";
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
import LoadableButton from "@/components/button/LoadableButton";
import Quota from "../Job/Interactive/Quota";
import PageTitle from "@/components/layout/PageTitle";
import { apiUserEmailVerified } from "@/services/api/user";
import { TimeDistance } from "@/components/custom/TimeDistance";

// Moved Zod schema to component
function getFormSchema(t: (key: string) => string) {
  return z.object({
    nickname: z.string().min(2, {
      message: t("userSettings.nicknameError"),
    }),
    email: z
      .string()
      .email({
        message: t("userSettings.emailError"),
      })
      .optional()
      .nullable(),
    teacher: z.string().optional().nullable(),
    group: z.string().optional().nullable(),
    expiredAt: z.string().optional().nullable(),
    phone: z.string().optional().nullable(),
    avatar: z.string().url().optional().nullable(),
  });
}

export default function UserSettings() {
  const { t } = useTranslation();
  const [user, setUser] = useAtom(globalUserInfo);
  const [avatarPreview, setAvatarPreview] = useState(user.avatar || "");
  const [originalEmail, setOriginalEmail] = useState(user.email || "");
  const [isEmailVerified, setIsEmailVerified] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isVerifyError, setIsVerifyError] = useState(false);
  const [verificationCode, setVerificationCode] = useState("");

  const formSchema = getFormSchema(t);

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

  const { data: emailVerified } = useQuery({
    queryKey: ["emailVerified"],
    queryFn: () => apiUserEmailVerified(),
    select: (res) => res.data,
  });

  const { mutate: updateUser } = useMutation({
    mutationFn: (values: IUserAttributes) =>
      apiContextUpdateUserAttributes(values),
    onSuccess: (_data, values) => {
      toast.success(t("userSettings.updateSuccess"));
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
      toast.success(t("userSettings.emailVerifiedSuccess"));
      setOriginalEmail(form.getValues("email") ?? "");
      setIsEmailVerified(true);
      setIsDialogOpen(false);
    },
  });

  useEffect(() => {
    if (emailVerified?.verified) {
      return;
    }
    setIsEmailVerified(false);
  }, [emailVerified]);

  function onSubmit(values: z.infer<typeof formSchema>) {
    if (!isEmailVerified && values.email !== originalEmail) {
      form.setError("email", {
        type: "manual",
        message: t("userSettings.verifyEmailFirst"),
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
      <div>
        <PageTitle
          title={t("userSettings.quotaTitle")}
          description={t("userSettings.quotaDescription")}
          className="mb-4"
        />
        <Quota />
      </div>
      <div>
        <PageTitle
          title={t("userSettings.userInfoTitle")}
          description={t("userSettings.userInfoDescription")}
          className="mb-4"
        />
        <Card className="lg:col-span-2">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)}>
              <CardContent className="space-y-8">
                <div className="flex flex-row items-center gap-6">
                  <FormField
                    control={form.control}
                    name="avatar"
                    render={({ field }) => (
                      <FormItem className="flex-1">
                        <FormLabel>{t("userSettings.avatarLabel")}</FormLabel>
                        <FormControl>
                          <div className="flex items-center space-x-4">
                            <Input
                              {...field}
                              value={field.value || ""}
                              placeholder={t("userSettings.avatarPlaceholder")}
                              className="font-mono"
                              onChange={(e) => {
                                field.onChange(e);
                                setAvatarPreview(e.target.value);
                              }}
                            />
                          </div>
                        </FormControl>
                        <FormDescription>
                          {t("userSettings.avatarDescription")}
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
                <FormField
                  control={form.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("userSettings.emailLabel")}</FormLabel>
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
                          {(field.value !== originalEmail ||
                            emailVerified?.verified !== true) && (
                            <LoadableButton
                              isLoading={isSendVerificationPending}
                              isLoadingText={t(
                                "userSettings.verificationLoading",
                              )}
                              variant="secondary"
                              type="button"
                              onClick={() => {
                                sendVerificationEmail(field.value || "");
                              }}
                            >
                              <MailPlusIcon />
                              {t("userSettings.verifyButton")}
                            </LoadableButton>
                          )}
                        </div>
                      </FormControl>
                      <FormDescription>
                        {t("userSettings.emailDescription")}
                        {emailVerified?.lastEmailVerifiedAt && (
                          <span className="ml-0.5">
                            ({t("userSettings.lastVerified")}{" "}
                            <TimeDistance
                              date={emailVerified.lastEmailVerifiedAt}
                            />
                            )
                          </span>
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
              <CardFooter className="px-6 pt-6">
                <Button type="submit">
                  <UserRoundCogIcon />
                  {t("userSettings.updateButton")}
                </Button>
              </CardFooter>
            </form>
          </Form>
          <AlertDialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>
                  {t("userSettings.verifyDialogTitle")}
                </AlertDialogTitle>

                {isVerifyError ? (
                  <AlertDialogDescription className="text-destructive">
                    {t("userSettings.invalidCode")}
                  </AlertDialogDescription>
                ) : (
                  <AlertDialogDescription>
                    {t("userSettings.verifyDialogDescription", {
                      email: form.getValues("email"),
                    })}
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
                <AlertDialogCancel>
                  {t("userSettings.cancelButton")}
                </AlertDialogCancel>
                <Button
                  onClick={() =>
                    verifyEmailCode({
                      code: verificationCode,
                      email: form.getValues("email") ?? "",
                    })
                  }
                >
                  {t("userSettings.verifyButton")}
                </Button>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </Card>
      </div>
    </>
  );
}
