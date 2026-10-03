/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { t } from 'i18next'
import { CircleHelpIcon } from 'lucide-react'
import type { ComponentProps } from 'react'

import {
  FormControl,
  FormDescription,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

interface TensorboardLogDirFormFieldProps {
  defaultPath: string
  descriptionKey?: string
  inputProps: ComponentProps<typeof Input>
}

export function TensorboardLogDirFormField({
  defaultPath,
  descriptionKey = 'tensorboard.jobForm.logDirDescription',
  inputProps,
}: TensorboardLogDirFormFieldProps) {
  const description = t(descriptionKey, { defaultPath })

  return (
    <FormItem>
      <div className="flex items-center gap-1">
        <FormLabel>{t('tensorboard.jobForm.logDirLabel')}</FormLabel>
        <TooltipProvider delayDuration={100}>
          <Tooltip>
            <TooltipTrigger asChild>
              <CircleHelpIcon className="text-muted-foreground size-4 cursor-help" />
            </TooltipTrigger>
            <TooltipContent className="max-w-sm">{description}</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
      <FormControl>
        <Input {...inputProps} className="font-mono" placeholder={defaultPath} />
      </FormControl>
      <FormDescription>{description}</FormDescription>
      <FormMessage />
    </FormItem>
  )
}
