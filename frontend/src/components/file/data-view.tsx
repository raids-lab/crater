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
import { Link, linkOptions } from '@tanstack/react-router'
import { BotIcon, DatabaseZapIcon, DownloadIcon, PackageIcon, PlusIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import DocsButton from '@/components/button/docs-button'
import ListedButton from '@/components/button/listed-button'
import { DataCreateForm } from '@/components/file/data-create-form'
import DataList from '@/components/layout/data-list'
import { ModelDownloadDialog } from '@/components/model/model-download-dialog'
import SandwichSheet from '@/components/sheet/sandwich-sheet'

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
    queryKey: ['data', sourceType || 'mydataset'],
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
  const [openDownloadSheet, setOpenDownloadSheet] = useState(false)

  const filteredData = data.data?.filter((dataset) => dataset.type === sourceType) || []

  return (
    <DataList
      items={
        filteredData.map((dataset) => ({
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
          <div className="flex min-w-0 items-center gap-3">
            <div
              className={`bg-primary/10 text-primary flex size-12 shrink-0 items-center justify-center rounded-lg p-2`}
            >
              {sourceTitle === '模型' ? (
                <BotIcon className="size-6" />
              ) : sourceTitle === '数据集' ? (
                <DatabaseZapIcon className="size-6" />
              ) : (
                <PackageIcon className="size-6" />
              )}
            </div>
            <TooltipLink
              {...getLinkOptions(sourceType || 'dataset')}
              params={{ id: `${item.id}` }}
              name={<p className="max-w-[400px] truncate text-left font-semibold">{item.name}</p>}
              tooltip={`查看${sourceTitle}详情`}
              className="min-w-0"
            />
          </div>
        )
      }}
      actionArea={
        <div className="flex flex-row gap-3">
          <Link
            to={
              sourceType === 'model'
                ? '/portal/data/models/downloads'
                : '/portal/data/datasets/downloads'
            }
          >
            <Button variant="outline" className="min-w-fit">
              <DownloadIcon className="size-4" />
              查看下载记录
            </Button>
          </Link>
          <DocsButton
            title={t('dataView.docsButtonTitle', { sourceTitle })}
            url={`file/${sourceType}`}
          />
          <ListedButton
            icon={<PlusIcon />}
            renderTitle={(title) => title || t('dataView.addButton', { sourceTitle })}
            itemTitle="操作"
            cacheKey={`${sourceType}-action`}
            items={[
              {
                key: 'download',
                title: sourceType === 'model' ? '下载模型' : '下载数据集',
                action: () => {
                  setOpenDownloadSheet(true)
                },
              },
              {
                key: 'add',
                title: t('dataView.addButton', { sourceTitle }),
                action: () => {
                  setOpenSheet(true)
                },
              },
            ]}
          />
          <SandwichSheet
            isOpen={openSheet}
            onOpenChange={setOpenSheet}
            title={t('dataView.createTitle', { sourceTitle })}
            description={t('dataView.createDescription', { sourceTitle })}
            className="sm:max-w-3xl"
          >
            <DataCreateForm closeSheet={() => setOpenSheet(false)} type={sourceType} />
          </SandwichSheet>
          <SandwichSheet
            isOpen={openDownloadSheet}
            onOpenChange={setOpenDownloadSheet}
            title={sourceType === 'model' ? '下载模型' : '下载数据集'}
            description={
              sourceType === 'model'
                ? '从 ModelScope 或 HuggingFace 下载模型到您的文件系统'
                : '从 ModelScope 或 HuggingFace 下载数据集到您的文件系统'
            }
            className="sm:max-w-3xl"
          >
            <ModelDownloadDialog
              closeSheet={() => setOpenDownloadSheet(false)}
              defaultCategory={sourceType as 'model' | 'dataset'}
            />
          </SandwichSheet>
        </div>
      }
    />
  )
}
