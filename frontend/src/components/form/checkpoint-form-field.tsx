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
import { SaveIcon } from 'lucide-react'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'

import {
  FormControl,
  FormDescription,
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
import { Switch } from '@/components/ui/switch'

import { cn } from '@/lib/utils'

import AccordionCard from './accordion-card'

interface CheckpointFormCardProps<T extends FieldValues> {
  form: UseFormReturn<T>
  basePath?: string
  open: boolean
  setOpen: (open: boolean) => void
}

const path = <T extends FieldValues>(basePath: string, key: string) =>
  `${basePath}.${key}` as FieldPath<T>

export function CheckpointFormCard<T extends FieldValues>({
  form,
  basePath = 'checkpoint',
  open,
  setOpen,
}: CheckpointFormCardProps<T>) {
  const enabled = form.watch(path<T>(basePath, 'enabled'))
  const resumeMode = form.watch(path<T>(basePath, 'resumeMode'))

  return (
    <AccordionCard cardTitle="Checkpoint" icon={SaveIcon} open={open} setOpen={setOpen}>
      <div className="mt-3 space-y-4">
        <FormField
          control={form.control}
          name={path<T>(basePath, 'enabled')}
          render={({ field }) => (
            <FormItem className="flex flex-row items-center justify-between space-y-0">
              <FormLabel className="font-normal">启用 checkpoint</FormLabel>
              <FormControl>
                <Switch checked={field.value} onCheckedChange={field.onChange} />
              </FormControl>
            </FormItem>
          )}
        />
        <div className={cn('space-y-4', !enabled && 'hidden')}>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField
              control={form.control}
              name={path<T>(basePath, 'framework')}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>框架</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="custom">Custom</SelectItem>
                      <SelectItem value="hf-trainer">HF Trainer</SelectItem>
                      <SelectItem value="pytorch">PyTorch</SelectItem>
                      <SelectItem value="deepspeed">DeepSpeed</SelectItem>
                      <SelectItem value="verl">VERL</SelectItem>
                      <SelectItem value="lightning">Lightning</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name={path<T>(basePath, 'resumeMode')}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>恢复策略</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="none">不恢复</SelectItem>
                      <SelectItem value="latest">Latest</SelectItem>
                      <SelectItem value="auto">Auto</SelectItem>
                      <SelectItem value="manual">指定 checkpoint</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField
              control={form.control}
              name={path<T>(basePath, 'projectName')}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>项目名</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name={path<T>(basePath, 'experimentName')}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>实验名</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name={path<T>(basePath, 'checkpointDir')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Checkpoint 目录</FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" placeholder="/workspace/checkpoints" />
                </FormControl>
                <FormDescription>留空时由平台选择第一个可写挂载路径生成默认目录</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={path<T>(basePath, 'outputDir')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Output 目录</FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" placeholder="/workspace/checkpoints" />
                </FormControl>
                <FormDescription>HF Trainer 等框架会将 output_dir 与 checkpoint 目录对齐</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          {resumeMode === 'manual' && (
            <FormField
              control={form.control}
              name={path<T>(basePath, 'resumeFrom')}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>恢复路径</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      className="font-mono"
                      placeholder="/workspace/checkpoints/checkpoint-1000"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}
          <div className="grid gap-3 sm:grid-cols-2">
            <FormField
              control={form.control}
              name={path<T>(basePath, 'saveSteps')}
              render={() => (
                <FormItem>
                  <FormLabel>保存间隔</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={1}
                      {...form.register(path<T>(basePath, 'saveSteps'), { valueAsNumber: true })}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name={path<T>(basePath, 'maxToKeep')}
              render={() => (
                <FormItem>
                  <FormLabel>保留数量</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min={1}
                      {...form.register(path<T>(basePath, 'maxToKeep'), { valueAsNumber: true })}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name={path<T>(basePath, 'maxBytes')}
            render={() => (
              <FormItem>
                <FormLabel>容量配额</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    min={0}
                    {...form.register(path<T>(basePath, 'maxBytes'), { valueAsNumber: true })}
                  />
                </FormControl>
                <FormDescription>单位为 Byte，0 表示不启用容量配额</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      </div>
    </AccordionCard>
  )
}
