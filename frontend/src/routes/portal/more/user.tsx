/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
// i18n-processed-v1.1.0
// Modified code
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useMutation } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { useAtom } from 'jotai'
import {
  BotIcon,
  Loader2Icon,
  MailPlusIcon,
  RotateCcwIcon,
  SaveIcon,
  UserRoundCogIcon,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { InputOTP, InputOTPGroup, InputOTPSeparator, InputOTPSlot } from '@/components/ui/input-otp'

import LoadableButton from '@/components/button/loadable-button'
import { TimeDistance } from '@/components/custom/time-distance'
import PageTitle from '@/components/layout/page-title'

import { IUserAttributes } from '@/services/api/admin/user'
import {
  apiContextGetLLMConfig,
  apiContextResetLLMConfig,
  apiContextUpdateLLMConfig,
  apiContextUpdateUserAttributes,
  apiSendVerificationEmail,
  apiVerifyEmailCode,
} from '@/services/api/context'
import { apiUserEmailVerified } from '@/services/api/user'

import { atomUserInfo } from '@/utils/store'

// Moved Zod schema to component
function getFormSchema(t: (key: string) => string) {
  return z.object({
    nickname: z.string().min(2, {
      message: t('userSettings.nicknameError'),
    }),
    email: z
      .string()
      .email({
        message: t('userSettings.emailError'),
      })
      .optional()
      .nullable(),
    teacher: z.string().optional().nullable(),
    group: z.string().optional().nullable(),
    expiredAt: z.string().optional().nullable(),
    phone: z.string().optional().nullable(),
    avatar: z.string().url().optional().nullable(),
  })
}

function getLLMFormSchema(t: (key: string, options?: Record<string, unknown>) => string) {
  return z.object({
    baseUrl: z.string().url({
      message: t('userSettings.llm.validation.url', {
        defaultValue: '请输入有效的 API 基础地址',
      }),
    }),
    modelName: z.string().min(1, {
      message: t('userSettings.llm.validation.model', {
        defaultValue: '模型名称不能为空',
      }),
    }),
    apiKey: z.string().min(1, {
      message: t('userSettings.llm.validation.apiKey', {
        defaultValue: '请输入 API 密钥',
      }),
    }),
  })
}

export const Route = createFileRoute('/portal/more/user')({
  component: RouteComponent,
})

function RouteComponent() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [user, setUser] = useAtom(atomUserInfo)
  const [avatarPreview, setAvatarPreview] = useState(user?.avatar || '')
  const [originalEmail, setOriginalEmail] = useState(user?.email || '')
  const [isEmailVerified, setIsEmailVerified] = useState(true)
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [isVerifyError, setIsVerifyError] = useState(false)
  const [verificationCode, setVerificationCode] = useState('')

  const formSchema = getFormSchema(t)
  const llmFormSchema = getLLMFormSchema(t)

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      nickname: user?.nickname,
      email: user?.email || null,
      teacher: user?.teacher || null,
      group: user?.group || null,
      expiredAt: user?.expiredAt || null,
      phone: user?.phone || null,
      avatar: user?.avatar || null,
    },
  })
  const llmForm = useForm<z.infer<typeof llmFormSchema>>({
    resolver: zodResolver(llmFormSchema),
    defaultValues: {
      baseUrl: '',
      modelName: '',
      apiKey: '',
    },
  })

  const { data: emailVerified } = useQuery({
    queryKey: ['emailVerified'],
    queryFn: () => apiUserEmailVerified(),
    select: (res) => res.data,
  })
  const { data: llmConfig } = useQuery({
    queryKey: ['context', 'llm'],
    queryFn: async () => (await apiContextGetLLMConfig()).data,
  })
  const { mutate: updateUser } = useMutation({
    mutationFn: (values: IUserAttributes) => apiContextUpdateUserAttributes(values),
    onSuccess: (_data, values) => {
      toast.success(t('userSettings.updateSuccess'))
      setUser((prev) => ({ ...prev, ...values, space: prev?.space || '' }))
    },
  })

  const { mutate: sendVerificationEmail, isPending: isSendVerificationPending } = useMutation({
    mutationFn: (email: string) => apiSendVerificationEmail(email),
    onSuccess: () => {
      setIsVerifyError(false)
      setVerificationCode('')
      setIsDialogOpen(true)
    },
  })

  const { mutate: verifyEmailCode } = useMutation({
    mutationFn: ({ email, code }: { email: string; code: string }) =>
      apiVerifyEmailCode(email, code),
    onError: () => {
      setIsVerifyError(true)
    },
    onSuccess: () => {
      toast.success(t('userSettings.emailVerifiedSuccess'))
      setOriginalEmail(form.getValues('email') ?? '')
      setIsEmailVerified(true)
      setIsDialogOpen(false)
    },
  })

  const { mutate: updateUserLLM, isPending: isUpdatingLLM } = useMutation({
    mutationFn: (values: z.infer<typeof llmFormSchema>) =>
      apiContextUpdateLLMConfig({
        baseUrl: values.baseUrl,
        modelName: values.modelName,
        apiKey: values.apiKey,
        validate: true,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['context', 'llm'] })
      await queryClient.invalidateQueries({ queryKey: ['agent-config-summary'] })
      toast.success(
        t('userSettings.llm.saveSuccess', {
          defaultValue: '个人模型服务配置已保存',
        })
      )
    },
  })

  const { mutate: resetUserLLM, isPending: isResettingLLM } = useMutation({
    mutationFn: apiContextResetLLMConfig,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['context', 'llm'] })
      await queryClient.invalidateQueries({ queryKey: ['agent-config-summary'] })
      toast.success(
        t('userSettings.llm.resetSuccess', {
          defaultValue: '已恢复使用平台默认模型服务配置',
        })
      )
    },
  })

  useEffect(() => {
    if (emailVerified?.verified) {
      return
    }
    setIsEmailVerified(false)
  }, [emailVerified])

  useEffect(() => {
    if (!llmConfig) return
    llmForm.reset({
      baseUrl: llmConfig.baseUrl || '',
      modelName: llmConfig.modelName || '',
      apiKey: llmConfig.apiKey || '',
    })
  }, [llmConfig, llmForm])

  function onSubmit(values: z.infer<typeof formSchema>) {
    if (!isEmailVerified && values.email !== originalEmail) {
      form.setError('email', {
        type: 'manual',
        message: t('userSettings.verifyEmailFirst'),
      })
      return
    }

    updateUser({
      id: user?.id || 0,
      name: user?.name || '',
      email: values.email ?? user?.email,
      nickname: values.nickname ?? user?.nickname,
      teacher: values.teacher ?? user?.teacher,
      group: values.group ?? user?.group,
      expiredAt: values.expiredAt ?? user?.expiredAt,
      phone: values.phone ?? user?.phone,
      avatar: values.avatar ?? user?.avatar,
    })
  }

  function onSubmitLLM(values: z.infer<typeof llmFormSchema>) {
    updateUserLLM(values)
  }

  return (
    <>
      <div>
        <PageTitle
          title={t('userSettings.userInfoTitle')}
          description={t('userSettings.userInfoDescription')}
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
                        <FormLabel>{t('userSettings.avatarLabel')}</FormLabel>
                        <FormControl>
                          <div className="flex items-center space-x-4">
                            <Input
                              {...field}
                              value={field.value || ''}
                              placeholder={t('userSettings.avatarPlaceholder')}
                              className="font-mono"
                              onChange={(e) => {
                                field.onChange(e)
                                setAvatarPreview(e.target.value)
                              }}
                            />
                          </div>
                        </FormControl>
                        <FormDescription>{t('userSettings.avatarDescription')}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Avatar className="h-20 w-20">
                    <AvatarImage src={avatarPreview} alt="Avatar preview" />
                    <AvatarFallback>
                      {user?.name?.charAt(0) || user?.nickname?.charAt(0) || '?'}
                    </AvatarFallback>
                  </Avatar>
                </div>
                <FormField
                  control={form.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('userSettings.emailLabel')}</FormLabel>
                      <FormControl>
                        <div className="flex items-center space-x-2">
                          <Input
                            {...field}
                            type="email"
                            className="font-mono"
                            value={field.value || ''}
                            onChange={(e) => {
                              field.onChange(e)
                              setIsEmailVerified(false)
                            }}
                          />
                          {(field.value !== originalEmail || emailVerified?.verified !== true) && (
                            <LoadableButton
                              isLoading={isSendVerificationPending}
                              isLoadingText={t('userSettings.verificationLoading')}
                              variant="secondary"
                              type="button"
                              onClick={() => {
                                sendVerificationEmail(field.value || '')
                              }}
                            >
                              <MailPlusIcon />
                              {t('userSettings.verifyButton')}
                            </LoadableButton>
                          )}
                        </div>
                      </FormControl>
                      <FormDescription>
                        {t('userSettings.emailDescription')}
                        {emailVerified?.lastEmailVerifiedAt && (
                          <span className="ml-0.5">
                            ({t('userSettings.lastVerified')}{' '}
                            <TimeDistance date={emailVerified.lastEmailVerifiedAt} />)
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
                  {t('userSettings.updateButton')}
                </Button>
              </CardFooter>
            </form>
          </Form>
          <AlertDialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('userSettings.verifyDialogTitle')}</AlertDialogTitle>

                {isVerifyError ? (
                  <AlertDialogDescription className="text-destructive">
                    {t('userSettings.invalidCode')}
                  </AlertDialogDescription>
                ) : (
                  <AlertDialogDescription>
                    {t('userSettings.verifyDialogDescription', {
                      email: form.getValues('email'),
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
                <AlertDialogCancel>{t('userSettings.cancelButton')}</AlertDialogCancel>
                <Button
                  onClick={() =>
                    verifyEmailCode({
                      code: verificationCode,
                      email: form.getValues('email') ?? '',
                    })
                  }
                >
                  {t('userSettings.verifyButton')}
                </Button>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </Card>
        <Card className="mt-4 lg:col-span-2">
          <CardHeader>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-2">
                <BotIcon className="text-primary h-5 w-5" />
                <CardTitle>
                  {t('userSettings.llm.title', {
                    defaultValue: '个人模型服务配置',
                  })}
                </CardTitle>
              </div>
              <Badge variant={llmConfig?.usingPersonal ? 'default' : 'secondary'}>
                {llmConfig?.usingPersonal
                  ? t('userSettings.llm.personalSource', { defaultValue: '个人配置生效中' })
                  : t('userSettings.llm.platformSource', { defaultValue: '平台默认配置' })}
              </Badge>
            </div>
            <CardDescription>
              {llmConfig?.usingPersonal
                ? t('userSettings.llm.usingPersonal', {
                    defaultValue:
                      '当前账号已启用个人模型服务配置，AI 功能将优先使用该配置进行调用。',
                  })
                : t('userSettings.llm.usingPlatform', {
                    defaultValue:
                      '当前账号使用平台默认模型服务配置。保存个人配置后，AI 功能将优先使用个人配置。',
                  })}
            </CardDescription>
          </CardHeader>
          <Form {...llmForm}>
            <form onSubmit={llmForm.handleSubmit(onSubmitLLM)}>
              <CardContent className="space-y-4">
                <FormField
                  control={llmForm.control}
                  name="baseUrl"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('userSettings.llm.baseUrl', {
                          defaultValue: 'API 基础地址',
                        })}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder="https://api.openai.com/v1" {...field} />
                      </FormControl>
                      <FormDescription>
                        {t('userSettings.llm.baseUrlDescription', {
                          defaultValue: '用于访问兼容 OpenAI Chat Completions 的模型服务端点。',
                        })}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={llmForm.control}
                  name="modelName"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('userSettings.llm.modelName', {
                          defaultValue: '模型名称',
                        })}
                      </FormLabel>
                      <FormControl>
                        <Input placeholder="gpt-4o / deepseek-chat" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={llmForm.control}
                  name="apiKey"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('userSettings.llm.apiKey', {
                          defaultValue: 'API 密钥',
                        })}
                      </FormLabel>
                      <FormControl>
                        <Input type="password" placeholder="sk-..." {...field} />
                      </FormControl>
                      <FormDescription>
                        {t('userSettings.llm.apiKeyDescription', {
                          defaultValue: '密钥仅用于当前账号的 AI 调用，保存后将加密存储。',
                        })}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
              <CardFooter className="flex flex-wrap justify-end gap-3 px-6 pt-6">
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    disabled={isUpdatingLLM || isResettingLLM || !llmConfig?.usingPersonal}
                    onClick={() => resetUserLLM()}
                  >
                    {isResettingLLM ? (
                      <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <RotateCcwIcon className="mr-2 h-4 w-4" />
                    )}
                    {t('userSettings.llm.resetButton', {
                      defaultValue: '恢复平台默认',
                    })}
                  </Button>
                  <Button type="submit" disabled={isUpdatingLLM || isResettingLLM}>
                    {isUpdatingLLM ? (
                      <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <SaveIcon className="mr-2 h-4 w-4" />
                    )}
                    {t('userSettings.llm.saveButton', {
                      defaultValue: '保存配置',
                    })}
                  </Button>
                </div>
              </CardFooter>
            </form>
          </Form>
        </Card>
      </div>
    </>
  )
}
