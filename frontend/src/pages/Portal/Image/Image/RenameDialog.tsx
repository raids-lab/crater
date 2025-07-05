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

import { type FC } from 'react'
import { Button } from '@/components/ui/button'
import {
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Check, Pencil, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'

interface RenameDialogProps {
  imageDescription: string
  onRename: (newDescription: string) => void
}

export const RenameDialog: FC<RenameDialogProps> = ({ imageDescription, onRename }) => {
  // 定义表单验证 schema
  const formSchema = z.object({
    newDescription: z
      .string()
      .min(1, '请输入有效的名称')
      .refine((value) => value !== imageDescription, {
        message: '新名称不能与当前名称相同',
      }),
  })

  type FormSchema = z.infer<typeof formSchema>

  // 使用 react-hook-form
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      newDescription: imageDescription,
    },
  })

  // 表单提交处理
  const onSubmit = (values: FormSchema) => {
    onRename(values.newDescription)
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle className="flex items-center gap-1 text-xl">
          <Pencil className="text-primary h-5 w-5" />
          <span>更新镜像名称</span>
        </DialogTitle>
      </DialogHeader>
      <DialogDescription className="space-y-4 pt-2">
        <div className="bg-muted/50 rounded-md px-4 py-3">
          <p className="text-muted-foreground text-sm">当前名称</p>
          <p className="mt-1 font-medium">「{imageDescription}」</p>
        </div>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-2">
            <FormField
              control={form.control}
              name="newDescription"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="font-medium">新名称</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder="输入新的描述" autoFocus />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>
      </DialogDescription>

      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline" className="flex items-center gap-2">
            <X className="h-4 w-4" />
            取消
          </Button>
        </DialogClose>
        <Button
          variant="default"
          className="flex items-center gap-2"
          onClick={form.handleSubmit(onSubmit)}
        >
          <Check className="h-4 w-4" />
          确认
        </Button>
      </DialogFooter>
    </>
  )
}
