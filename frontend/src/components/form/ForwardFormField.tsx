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
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
import { useTranslation } from 'react-i18next'
import { Fragment, useState } from 'react'
import { ArrayPath, UseFormReturn, useFieldArray } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { CirclePlus, NetworkIcon, XIcon } from 'lucide-react'
import FormLabelMust from '@/components/form/FormLabelMust'
import AccordionCard from '@/components/form/AccordionCard'
import { cn } from '@/lib/utils'

const getForwardCardTitle = (t) => t('forwardCard.title')

interface ForwardFormCardProps<
  T extends {
    forwards: Array<{ name: string; port: number }>
  },
> {
  form: UseFormReturn<T>
  className?: string
}

export function ForwardFormCard<
  T extends {
    forwards: Array<{ name: string; port: number }>
  },
>({ form, className }: ForwardFormCardProps<T>) {
  const { t } = useTranslation()
  const [forwardOpen, setForwardOpen] = useState<boolean>(true)

  // Field array for forwards
  const {
    fields: forwardFields,
    append: forwardAppend,
    remove: forwardRemove,
  } = useFieldArray({
    name: 'forwards' as ArrayPath<T>,
    control: form.control,
  })

  return (
    <AccordionCard
      cardTitle={getForwardCardTitle(t)}
      icon={NetworkIcon}
      open={forwardOpen}
      setOpen={setForwardOpen}
      className={className}
    >
      <div className="mt-3 space-y-5">
        {forwardFields.map((field, index) => (
          <Fragment key={field.id}>
            <Separator className={cn('mb-5', index === 0 && 'hidden')} />
            <div key={field.id} className="relative">
              <div className="space-y-5">
                <FormField
                  control={form.control}
                  name={`forwards.${index}.name`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('forwardForm.nameLabel', { index: index + 1 })}
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormDescription>{t('forwardForm.nameDescription')}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`forwards.${index}.port`}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('forwardForm.portLabel', { index: index + 1 })}
                        <FormLabelMust />
                      </FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type="number"
                          onChange={(e) => {
                            const value = e.target.value
                            if (value === '') {
                              field.onChange(null)
                            } else {
                              const parsed = parseInt(value, 10)
                              if (!isNaN(parsed)) {
                                field.onChange(parsed)
                              }
                            }
                          }}
                          value={field.value ?? ''}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <button
                  type="button"
                  onClick={() => forwardRemove(index)}
                  className="data-[state=open]:bg-accent data-[state=open]:text-muted-foreground absolute -top-1.5 right-0 cursor-pointer rounded-sm opacity-50 transition-opacity hover:opacity-100 focus:outline-hidden disabled:pointer-events-none"
                >
                  <XIcon className="h-4 w-4" />
                  <span className="sr-only">{t('forwardForm.removeButton')}</span>
                </button>
              </div>
            </div>
          </Fragment>
        ))}
        <Button
          type="button"
          variant="secondary"
          className="w-full"
          onClick={() =>
            forwardAppend({
              name: '',
              port: null, // Set port to null for no default value
            })
          }
        >
          <CirclePlus className="size-4" />
          {t('forwardForm.addButton')}
        </Button>
      </div>
    </AccordionCard>
  )
}
