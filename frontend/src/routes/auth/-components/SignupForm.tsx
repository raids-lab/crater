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
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { isAxiosError } from 'axios'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
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

import LoadableButton from '@/components/button/loadable-button'

import { apiUserSignup } from '@/services/api/auth'
import { ERROR_REGISTER_NOT_FOUND, ERROR_REGISTER_TIMEOUT } from '@/services/error_code'
import { IErrorResponse } from '@/services/types'

const formSchema = z
  .object({
    username: z
      .string()
      .min(1, {
        message: 'Username can not be empty.',
      })
      .max(20, {
        message: 'Username must be at most 20 characters.',
      }),
    password: z
      .string()
      .min(1, {
        message: 'Password can not be empty.',
      })
      .max(20, {
        message: 'Password must be at most 20 characters.',
      }),
    passwordConfirm: z.string(),
  })
  .refine((data) => data.password === data.passwordConfirm, {
    message: "Passwords don't match",
    path: ['passwordConfirm'],
  })

export function SignupForm() {
  const navigate = useNavigate()
  const { mutate: loginUser, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiUserSignup({
        userName: values.username,
        password: values.password,
      }),
    onSuccess: () => {
      toast.success('注册成功')
      navigate({ to: '/auth', search: { redirect: '/', token: '' } })
    },
    onError: (error) => {
      if (isAxiosError<IErrorResponse>(error)) {
        const errorCode = error.response?.data.code
        switch (errorCode) {
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

  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      username: '',
      password: '',
      passwordConfirm: '',
    },
  })

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    if (isPending) {
      return
    }
    loginUser(values)
  }

  return (
    <Form {...form}>
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
        <LoadableButton
          isLoadingText="注册中"
          type="submit"
          className="w-full"
          isLoading={isPending}
        >
          注册
        </LoadableButton>
      </form>
    </Form>
  )
}
