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
import { useTranslation } from 'react-i18next'

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { IDataset } from '@/services/api/dataset'

import TipBadge from '../badge/TipBadge'
import { TimeDistance } from '../custom/TimeDistance'
import { ComboboxItem } from './Combobox'

export default function DatasetItem({ item }: { item: ComboboxItem<IDataset> }) {
  const { t } = useTranslation()

  const dataType = item.detail?.type
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex w-full flex-col items-start gap-1">
            <div className="flex w-full flex-row items-center gap-1.5">
              <p>{item.detail?.name}</p>
            </div>
            {item.detail?.describe && (
              <p className="text-muted-foreground text-xs" data-description>
                {item.detail?.describe}
              </p>
            )}
            <div className="mt-1 flex flex-wrap gap-1">
              <TipBadge
                className="text-highlight-slate bg-highlight-slate/15"
                title={dataType === 'model' ? t('datasetItem.model') : t('datasetItem.dataset')}
              />
              {item.detail?.extra.tag?.map((tag) => (
                <TipBadge
                  key={tag}
                  title={tag}
                  className="text-highlight-slate bg-highlight-slate/15"
                />
              ))}
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent className="flex flex-row gap-0.5">
          <p>{item.detail?.userInfo.nickname}</p>
          {t('datasetItem.createdOn')}
          <TimeDistance date={item.detail?.createdAt} className="text-xs" />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
