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
import { CircleArrowUp } from 'lucide-react'
import { FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { exportToJsonFile } from '@/utils/form'
import { showErrorToast } from '@/utils/toast'

import TooltipButton from '../button/tooltip-button'
import { MetadataFormType } from './types'

interface ImportButtonProps<T extends FieldValues> {
  metadata: MetadataFormType
  form?: UseFormReturn<T>
  buttonText?: string
  className?: string
}

function FormExportButton<T extends FieldValues>({
  metadata,
  form,
  className,
  buttonText,
}: ImportButtonProps<T>) {
  const { t } = useTranslation()

  const currentValues = form?.getValues()
  return (
    <TooltipButton
      variant="outline"
      type="button"
      className={className}
      tooltipContent={t('formExportButton.tooltipContent')}
      onClick={() => {
        form
          ?.trigger()
          .then((isValid) => {
            if (!isValid) {
              return
            }
            // 导出为小写 formType + 当前日期 MMDD + .json
            // 例如：job-0910.json
            const fileName = `${metadata.type.toLowerCase()}_${new Date().toLocaleDateString(
              'en-US',
              {
                month: '2-digit',
                day: '2-digit',
              }
            )}.json`
            exportToJsonFile(
              {
                version: metadata.version,
                type: metadata.type,
                data: currentValues,
              },
              fileName
            )
          })
          .catch((error) => {
            showErrorToast(error)
          })
      }}
    >
      <CircleArrowUp className="size-4" />
      {buttonText || t('formExportButton.buttonText')}
    </TooltipButton>
  )
}

export default FormExportButton
