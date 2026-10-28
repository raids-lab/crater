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

import { logger } from '@/utils/loglevel'

const formSchema = z.object({
  email: z
    .string()
    .min(1, {
      message: '邮箱不能为空',
    })
    .email({
      message: '请输入有效的邮箱地址',
    }),
})

export function ForgotPasswordForm() {
  // 1. Define your form.
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: '',
    },
  })

  const { mutate: sendResetEmail, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) => {
      // 这里添加发送重置密码邮件的API调用
      // 目前只是模拟API调用
      logger.info('Sending reset email', values)
      return new Promise<void>((resolve) => {
        setTimeout(() => resolve(), 1000)
      })
    },
    onSuccess: () => {
      toast.success('重置密码邮件已发送，请查收')
      form.reset()
    },
    onError: () => {
      toast.error('发送邮件失败，请稍后重试')
    },
  })

  // 2. Define a submit handler.
  const onSubmit = (values: z.infer<typeof formSchema>) => {
    sendResetEmail(values)
  }

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
  )
}
