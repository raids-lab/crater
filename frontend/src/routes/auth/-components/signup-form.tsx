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

import { apiSignup } from '@/services/api/auth'
import { getAuthErrorMessage } from '@/services/auth-errors'
import { authMutationOptions } from '@/services/auth-mutation'
import { authText } from '@/services/auth-text'
import { getApiErrorData, markApiErrorHandled } from '@/services/client'
import { ERROR_RESOURCE_ALREADY_EXISTS } from '@/services/error_code'

const formSchema = z
  .object({
    username: z
      .string()
      .min(1, {
        message: '用户名不能为空',
      })
      .max(64, { message: authText('usernameTooLong') })
      .regex(/^[a-zA-Z0-9][a-zA-Z0-9._-]*$/, { message: authText('usernameFormat') }),
    password: z
      .string()
      .min(1, {
        message: '密码不能为空',
      })
      .refine((value) => new TextEncoder().encode(value).length <= 72, {
        message: authText('passwordTooLong'),
      }),
    passwordConfirm: z.string(),
  })
  .refine((data) => data.password === data.passwordConfirm, {
    message: '两次输入的密码不一致',
    path: ['passwordConfirm'],
  })

export function SignupForm({ onSuccess }: { onSuccess: () => void }) {
  const { mutate: loginUser, isPending } = useMutation({
    ...authMutationOptions,
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiSignup({
        userName: values.username,
        password: values.password,
      }),
    onSuccess: () => {
      toast.success('注册成功')
      onSuccess()
    },
    onError: (error) => {
      markApiErrorHandled(error)
      if (getApiErrorData(error)?.code === ERROR_RESOURCE_ALREADY_EXISTS) {
        form.setError('username', {
          type: 'server',
          message: getAuthErrorMessage(error),
        })
        return
      }
      form.setError('root.server', { type: 'server', message: getAuthErrorMessage(error) })
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
        {form.formState.errors.root?.server?.message && (
          <p role="alert" className="text-destructive text-sm">
            {form.formState.errors.root.server.message}
          </p>
        )}
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
