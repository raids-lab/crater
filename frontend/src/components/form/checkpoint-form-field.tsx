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
import { useTranslation } from 'react-i18next'

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
  const { t } = useTranslation()
  const enabled = form.watch(path<T>(basePath, 'enabled'))
  const resumeMode = form.watch(path<T>(basePath, 'resumeMode'))

  return (
    <AccordionCard cardTitle={t('checkpoint.title')} icon={SaveIcon} open={open} setOpen={setOpen}>
      <div className="mt-3 space-y-4">
        <FormField
          control={form.control}
          name={path<T>(basePath, 'enabled')}
          render={({ field }) => (
            <FormItem className="flex flex-row items-center justify-between space-y-0">
              <FormLabel className="font-normal">{t('checkpoint.form.enabled')}</FormLabel>
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
                  <FormLabel>{t('checkpoint.form.framework')}</FormLabel>
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
                  <FormLabel>{t('checkpoint.form.resumeMode')}</FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="none">{t('checkpoint.form.resumeModes.none')}</SelectItem>
                      <SelectItem value="latest">Latest</SelectItem>
                      <SelectItem value="auto">Auto</SelectItem>
                      <SelectItem value="manual">
                        {t('checkpoint.form.resumeModes.manual')}
                      </SelectItem>
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
                  <FormLabel>{t('checkpoint.form.projectName')}</FormLabel>
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
                  <FormLabel>{t('checkpoint.form.experimentName')}</FormLabel>
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
                <FormLabel>{t('checkpoint.form.checkpointDir')}</FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" placeholder="/workspace/checkpoints" />
                </FormControl>
                <FormDescription>{t('checkpoint.form.checkpointDirDescription')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={path<T>(basePath, 'outputDir')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('checkpoint.form.outputDir')}</FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" placeholder="/workspace/checkpoints" />
                </FormControl>
                <FormDescription>{t('checkpoint.form.outputDirDescription')}</FormDescription>
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
                  <FormLabel>{t('checkpoint.form.resumeFrom')}</FormLabel>
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
                  <FormLabel>{t('checkpoint.form.saveSteps')}</FormLabel>
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
                  <FormLabel>{t('checkpoint.form.maxToKeep')}</FormLabel>
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
                <FormLabel>{t('checkpoint.form.maxBytes')}</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    min={0}
                    {...form.register(path<T>(basePath, 'maxBytes'), { valueAsNumber: true })}
                  />
                </FormControl>
                <FormDescription>{t('checkpoint.form.maxBytesDescription')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      </div>
    </AccordionCard>
  )
}
