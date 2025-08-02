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
import { useQuery } from '@tanstack/react-query'
import { linkOptions } from '@tanstack/react-router'
import { BotIcon, DatabaseZapIcon, PackageIcon, PlusIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import DocsButton from '@/components/button/docs-button'
import { DataCreateForm } from '@/components/file/data-create-form'
import DataList from '@/components/layout/data-list'
import SandwichSheet from '@/components/sheet/SandwichSheet'

import { IDataset } from '@/services/api/dataset'
import { IResponse } from '@/services/types'

import TooltipLink from '../label/tooltip-link'

interface DatesetTableProps {
  sourceType?: 'dataset' | 'model' | 'sharefile'
  apiGetDataset: () => Promise<IResponse<IDataset[]>>
}

const getLinkOptions = (sourceType: string) => {
  switch (sourceType) {
    case 'model':
      return linkOptions({ to: '/portal/data/models/$id', search: { tab: '' }, params: { id: '' } })
    case 'sharefile':
      return linkOptions({ to: '/portal/data/blocks/$id', search: { tab: '' }, params: { id: '' } })
    default:
      return linkOptions({
        to: '/portal/data/datasets/$id',
        search: { tab: '' },
        params: { id: '' },
      })
  }
}

export function DataView({ apiGetDataset, sourceType }: DatesetTableProps) {
  const { t } = useTranslation()
  const data = useQuery({
    queryKey: ['data', 'mydataset'],
    queryFn: () => apiGetDataset(),
    select: (res) => res.data,
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
      mainArea={(item) => {
        return (
          <div className="flex items-center gap-2">
            <div
              className={`bg-primary/10 text-primary flex size-10 items-center justify-center rounded-lg p-1`}
            >
              {sourceTitle === '模型' ? (
                <BotIcon />
              ) : sourceTitle === '数据集' ? (
                <DatabaseZapIcon />
              ) : (
                <PackageIcon />
              )}
            </div>
            <TooltipLink
              {...getLinkOptions(sourceType || 'dataset')}
              params={{ id: `${item.id}` }}
              name={<p className="text-left">{item.name}</p>}
              tooltip={`查看${sourceTitle}详情`}
              className="font-semibold"
            />
          </div>
        )
      }}
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
