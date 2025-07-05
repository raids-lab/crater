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
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { IDataset } from '@/services/api/dataset'
import DocsButton from '@/components/button/DocsButton'
import { DataCreateForm } from '@/pages/Portal/Data/CreateForm'
import { IResponse } from '@/services/types'
import DataList from '@/pages/Portal/Data/DataList'
import { AxiosResponse } from 'axios'
import SandwichSheet from '@/components/sheet/SandwichSheet'
import { PlusIcon } from 'lucide-react'

interface DatesetTableProps {
  sourceType?: 'dataset' | 'model' | 'sharefile'
  apiGetDataset: () => Promise<AxiosResponse<IResponse<IDataset[]>>>
}

export function DataView({ apiGetDataset, sourceType }: DatesetTableProps) {
  const { t } = useTranslation()
  const data = useQuery({
    queryKey: ['data', 'mydataset'],
    queryFn: () => apiGetDataset(),
    select: (res) => res.data.data,
  })

  const sourceTypeMap = {
    model: t('dataView.sourceType.model'),
    sharefile: t('dataView.sourceType.sharefile'),
    dataset: t('dataView.sourceType.dataset'),
  }

  const sourceTitle = sourceType ? sourceTypeMap[sourceType] : sourceTypeMap.dataset

  const [openSheet, setOpenSheet] = useState(false)

  return (
    <DataList
      items={
        data.data
          ?.filter((dataset) => dataset.type === sourceType)
          .map((dataset) => ({
            id: dataset.id,
            name: dataset.name,
            desc: dataset.describe,
            tag: dataset.extra.tag || [],
            createdAt: dataset.createdAt,
            owner: dataset.userInfo,
          })) || []
      }
      title={sourceTitle}
      actionArea={
        <div className="flex flex-row gap-3">
          <DocsButton
            title={t('dataView.docsButtonTitle', { sourceTitle })}
            url={`file/${sourceType}`}
          />
          <SandwichSheet
            isOpen={openSheet}
            onOpenChange={setOpenSheet}
            title={t('dataView.createTitle', { sourceTitle })}
            description={t('dataView.createDescription', { sourceTitle })}
            trigger={
              <Button className="min-w-fit">
                <PlusIcon />
                {t('dataView.addButton', { sourceTitle })}
              </Button>
            }
            className="sm:max-w-3xl"
          >
            <DataCreateForm closeSheet={() => setOpenSheet(false)} type={sourceType} />
          </SandwichSheet>
        </div>
      }
    />
  )
}
