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
import { Plus, X } from 'lucide-react'
import { type FC } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { DialogClose, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { TaintEffect } from '@/services/api/cluster'

import FormLabelMust from '../form/FormLabelMust'
import { DialogType } from './node-mark'

interface AddNodeMarkDialogProps {
  type: DialogType
  onAdd: (key: string, value: string, effect?: string) => void
}

// 定义表单 schema
const createFormSchema = (type: DialogType) => {
  const keySchema = z
    .string()
    .min(1, '键名不能为空')
    .max(253, '键名最多253个字符')
    .refine((value) => /^[a-zA-Z0-9.-]+$/.test(value), '键名只能包含字母、数字、点和横线')

  // 当type为taint时，value可以为空
  const valueSchema =
    type === DialogType.TAINT
      ? z.string().max(63, '值最多63个字符').optional().or(z.literal(''))
      : z.string().min(1, '值不能为空').max(63, '值最多63个字符')

  const baseSchema = z.object({
    key: keySchema,
    value: valueSchema,
  })

  if (type === DialogType.TAINT) {
    return baseSchema.extend({
      effect: z.enum(Object.values(TaintEffect) as [string, ...string[]], {
        required_error: '请选择Effect',
      }),
    })
  }

  return baseSchema
}

type FormValues = {
  key: string
  value: string
  effect?: TaintEffect
}

const TYPE_CONFIG = {
  label: {
    title: '添加Label',
    keyLabel: 'Key',
    valueLabel: 'Value',
    keyPlaceholder: '请输入Label Key',
    valuePlaceholder: '请输入Label Value',
  },
  annotation: {
    title: '添加Annotation',
    keyLabel: 'Key',
    valueLabel: 'Value',
    keyPlaceholder: '请输入Annotation Key',
    valuePlaceholder: '请输入Annotation Value',
  },
  taint: {
    title: '添加Taint',
    keyLabel: 'Key',
    valueLabel: 'Value',
    keyPlaceholder: '请输入Taint Key',
    valuePlaceholder: '请输入Taint Value',
  },
} as const

export const AddNodeMarkDialog: FC<AddNodeMarkDialogProps> = ({ type, onAdd }) => {
  const config = TYPE_CONFIG[type]
  const needsEffect = type === 'taint'
  const formSchema = createFormSchema(type)

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      key: '',
      value: '',
      effect: needsEffect ? undefined : undefined,
    },
  })

  const handleSubmit = (values: FormValues) => {
    if (needsEffect && values.effect) {
      onAdd(values.key, values.value, values.effect)
    } else {
      onAdd(values.key, values.value)
    }
    form.reset()
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2 text-xl">
          <Plus className="text-primary h-5 w-5" />
          <span>{config.title}</span>
        </DialogTitle>
      </DialogHeader>

      <div className="space-y-4 pt-2">
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="key"
              render={({ field }) => (
                <FormItem className="w-full">
                  <FormLabel className="text-foreground">
                    {config.keyLabel}
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder={config.keyPlaceholder}
                      {...field}
                      className="text-foreground w-full"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="value"
              render={({ field }) => (
                <FormItem className="w-full">
                  <FormLabel className="text-foreground">
                    {config.valueLabel}
                    {type !== DialogType.TAINT && <FormLabelMust />}
                  </FormLabel>
                  <FormControl>
                    <Input
                      placeholder={config.valuePlaceholder}
                      {...field}
                      className="text-foreground w-full"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {needsEffect && (
              <FormField
                control={form.control}
                name="effect"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel className="text-foreground">
                      Effect
                      <FormLabelMust />
                    </FormLabel>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <FormControl>
                        <SelectTrigger className="text-foreground w-full">
                          <SelectValue placeholder="请选择Effect" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {Object.values(TaintEffect).map((effectOption) => (
                          <SelectItem key={effectOption} value={effectOption}>
                            {effectOption}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </form>
        </Form>
      </div>

      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline" className="flex items-center gap-2">
            <X className="size-4" />
            取消
          </Button>
        </DialogClose>

        <DialogClose asChild>
          <Button
            onClick={form.handleSubmit(handleSubmit)}
            disabled={!form.formState.isValid}
            className="flex items-center gap-2"
          >
            <Plus className="size-4" />
            确认添加
          </Button>
        </DialogClose>
      </DialogFooter>
    </>
  )
}
