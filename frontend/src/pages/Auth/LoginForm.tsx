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

import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useForm } from 'react-hook-form'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAtomValue, useSetAtom } from 'jotai'
import { globalLastView, globalAccount, globalUserInfo, useResetStore } from '@/utils/store'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Role, apiUserLogin } from '@/services/api/auth'
import LoadableButton from '@/components/button/LoadableButton'
import { toast } from 'sonner'
import { useEffect, useState } from 'react'
import { isAxiosError } from 'axios'
import { IErrorResponse } from '@/services/types'
import {
  ERROR_INVALID_CREDENTIALS,
  ERROR_MUST_REGISTER,
  ERROR_REGISTER_NOT_FOUND,
  ERROR_REGISTER_TIMEOUT,
} from '@/services/error_code'
import { ExternalLink } from 'lucide-react'
import { AuthMode } from '.'
import { configUrlWebsiteBaseAtom } from '@/utils/store/config'

const formSchema = z.object({
  username: z
    .string()
    .min(1, {
      message: '用户名不能为空',
    })
    .max(20, {
      message: '用户名最多 20 个字符',
    })
    .refine(
      (value) => {
        // 首字符必须小写字母，包含小写字母、数字、中划线
        const regex = /^[a-z][a-z0-9-]*[a-z0-9]$/
        return regex.test(value)
      },
      {
        message: '只能包含小写字母和数字，中划线可作为连接符',
      }
    ),
  password: z
    .string()
    .min(1, {
      message: '密码不能为空',
    })
    .max(20, {
      message: '密码最多 20 个字符',
    }),
})

interface LoginFormProps {
  authMode: AuthMode
  onForgotPasswordClick: () => void
}

export function LoginForm({ authMode, onForgotPasswordClick }: LoginFormProps) {
  const [open, setOpen] = useState(false)
  const location = useLocation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const setUserState = useSetAtom(globalUserInfo)
  const setAccount = useSetAtom(globalAccount)
  const { resetAll } = useResetStore()
  const lastView = useAtomValue(globalLastView)
  const website = useAtomValue(configUrlWebsiteBaseAtom)

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: '',
      password: '',
    },
  })

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (status !== 'pending') {
      resetAll()
      loginUser({
        username: values.username,
        password: values.password,
        auth: authMode == AuthMode.ACT ? 'act-ldap' : 'normal',
      })
    }
  }

  const { mutate: loginUser, status } = useMutation({
    mutationFn: (values: { auth: string; username?: string; password?: string; token?: string }) =>
      apiUserLogin({
        auth: values.auth,
        username: values.username,
        password: values.password,
        token: values.token,
      }),
    onSuccess: async (data) => {
      await queryClient.invalidateQueries()
      setUserState({
        ...data.user,
        space: data.context.space,
      })
      setAccount(data.context)
      toast.success(
        `你好，${data.context.rolePlatform ? '系统管理员' : '用户'}${data.user.nickname}`
      )
      // navigate to /portal and clear all history
      const dashboard =
        lastView === 'admin' && data.context.rolePlatform === Role.Admin
          ? '/admin'
          : '/portal/overview'
      navigate(dashboard, { replace: true })
    },
    onError: (error) => {
      if (isAxiosError<IErrorResponse>(error)) {
        const errorCode = error.response?.data.code
        switch (errorCode) {
          case ERROR_INVALID_CREDENTIALS:
            form.setError('password', {
              type: 'manual',
              message: '用户名或密码错误',
            })
            return
          case ERROR_MUST_REGISTER:
            setOpen(true)
            return
          case ERROR_REGISTER_TIMEOUT:
            toast.error('新用户注册访问 UID Server 超时，请联系管理员')
            return
          case ERROR_REGISTER_NOT_FOUND:
            toast.error('新用户注册访问 UID Server 失败，请联系管理员')
            return
        }
      } else {
        toast.error('登录失败，请稍后重试')
      }
    },
  })

  useEffect(() => {
    const searchParams = new URLSearchParams(location.search)
    const token = searchParams.get('token')
    if (token && token.length > 0) {
      loginUser({
        auth: 'act-api',
        token,
      })
    }
  }, [location, loginUser])

  return (
    <>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <FormField
            control={form.control}
            name="username"
            render={({ field }) => (
              <FormItem>
                <FormLabel>账号</FormLabel>
                <FormControl>
                  <Input autoComplete="username" {...field} />
                </FormControl>
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
                    className="text-muted-foreground p-0 text-sm underline"
                    type="button"
                    onClick={onForgotPasswordClick}
                  >
                    忘记密码？
                  </button>
                </div>
                <FormControl>
                  <Input type="password" autoComplete="current-password" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <LoadableButton
            isLoadingText="登录中"
            type="submit"
            className="w-full"
            isLoading={status === 'pending'}
          >
            {authMode === AuthMode.ACT ? 'ACT 认证登录' : '登录'}
          </LoadableButton>
        </form>
      </Form>
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>账号未激活</AlertDialogTitle>
            <AlertDialogDescription>
              第一次登录平台时，需要从 ACT 门户同步用户信息，请参考「
              <a href={`${website}/docs/user/quick-start/login`}>平台访问指南</a>
              」激活您的账号。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => window.open(`${website}/docs/user/quick-start/login`)}
            >
              <ExternalLink />
              立即阅读
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
