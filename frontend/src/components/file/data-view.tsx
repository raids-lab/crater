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
import { ArrowLeftIcon, DownloadIcon, PlusIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import DocsButton from '@/components/button/docs-button'
import ListedButton from '@/components/button/listed-button'
import { DataCreateForm } from '@/components/file/data-create-form'
import DataList from '@/components/layout/data-list'
import { ModelDownloadDialog } from '@/components/model/model-download-dialog'
import RepositorySourceMark from '@/components/model/repository-source-mark'
import SandwichSheet from '@/components/sheet/sandwich-sheet'

import { IDataset } from '@/services/api/dataset'
import { IResponse } from '@/services/types'

import TooltipLink from '../label/tooltip-link'

interface DatesetTableProps {
  sourceType?: 'dataset' | 'model' | 'sharefile'
  apiGetDataset: () => Promise<IResponse<IDataset[]>>
  organization?: string
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

export function DataView({ apiGetDataset, sourceType, organization }: DatesetTableProps) {
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

  const filteredData = useMemo(
    () =>
      data.data?.filter(
        (dataset) =>
          dataset.type === sourceType &&
          (!organization ||
            (dataset.organization || dataset.name.split('/')[0]).toLowerCase() ===
              organization.toLowerCase())
      ) || [],
    [data.data, organization, sourceType]
  )
  const isShareFile = sourceType === 'sharefile'

  const resourceItems = filteredData.map((dataset) => ({
    id: dataset.id,
    name: dataset.name,
    desc: dataset.describe,
    tag: (dataset.extra.tag || []).filter((tag) => tag !== 'auto-download').slice(0, 4),
    createdAt: dataset.createdAt,
    updatedAt: dataset.updatedAt,
    sourceUpdatedAt: dataset.sourceUpdatedAt,
    mountCount: dataset.mountCount,
    sizeBytes: dataset.sizeBytes,
    downloadCount: dataset.downloadCount,
    likes: dataset.likes,
    source: dataset.source,
    organization: dataset.organization,
    organizationUrl: dataset.organizationUrl,
    owner: dataset.userInfo,
  }))

  const organizationItems = useMemo(() => {
    if (sourceType !== 'model' || organization) {
      return []
    }
    const groups = new Map<string, { name: string; models: IDataset[] }>()
    for (const model of filteredData) {
      const name = model.organization || model.name.split('/')[0] || '其他'
      const key = name.toLowerCase()
      const group = groups.get(key)
      if (group) {
        group.models.push(model)
        if (group.name === group.name.toLowerCase() && name !== name.toLowerCase()) {
          group.name = name
        }
      } else {
        groups.set(key, { name, models: [model] })
      }
    }
    return Array.from(groups.values()).map(({ name, models }) => {
      const latest = [...models].sort(
        (a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
      )[0]
      const logoModel = models.find((model) => model.organizationUrl) || latest
      const sources = Array.from(
        new Set(
          models
            .flatMap((model) => [model.source, ...(model.extra.tag || [])])
            .filter((tag) => tag === 'huggingface' || tag === 'modelscope')
        )
      )
      return {
        id: Math.min(...models.map((model) => model.id)),
        name,
        desc: `${models.length} 个模型`,
        tag: sources,
        searchTerms: models.map((model) => model.name),
        createdAt: latest.createdAt,
        updatedAt: latest.updatedAt,
        sourceUpdatedAt: latest.sourceUpdatedAt,
        mountCount: models.reduce((sum, model) => sum + (model.mountCount || 0), 0),
        sizeBytes: models.reduce((sum, model) => sum + (model.sizeBytes || 0), 0),
        downloadCount: models.reduce((sum, model) => sum + (model.downloadCount || 0), 0),
        likes: models.reduce((sum, model) => sum + (model.likes || 0), 0),
        source: sources.length === 1 ? sources[0] : undefined,
        organization: name,
        organizationUrl: logoModel.organizationUrl,
        owner: latest.userInfo,
      }
    })
  }, [filteredData, organization, sourceType])

  const actionArea = (
    <div className="flex w-full flex-wrap gap-2 sm:w-auto sm:flex-nowrap sm:gap-3">
      {organization && sourceType === 'model' && (
        <Link to="/portal/data/models" search={{ organization: undefined }}>
          <Button variant="outline" className="min-w-fit">
            <ArrowLeftIcon className="size-4" />
            全部组织
          </Button>
        </Link>
      )}
      {!isShareFile && (
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
      )}
      <DocsButton
        title={t('dataView.docsButtonTitle', { sourceTitle })}
        url={`file/${sourceType}`}
      />
      {isShareFile ? (
        <Button onClick={() => setOpenSheet(true)}>
          <PlusIcon className="size-4" />
          {t('dataView.addButton', { sourceTitle })}
        </Button>
      ) : (
        <ListedButton
          icon={<PlusIcon />}
          renderTitle={(title) => title || t('dataView.addButton', { sourceTitle })}
          itemTitle="操作"
          cacheKey={`${sourceType}-action`}
          items={[
            {
              key: 'download',
              title: sourceType === 'model' ? '下载模型' : '下载数据集',
              action: () => setOpenDownloadSheet(true),
            },
            {
              key: 'add',
              title: t('dataView.addButton', { sourceTitle }),
              action: () => setOpenSheet(true),
            },
          ]}
        />
      )}
      <SandwichSheet
        isOpen={openSheet}
        onOpenChange={setOpenSheet}
        title={t('dataView.createTitle', { sourceTitle })}
        description={t('dataView.createDescription', { sourceTitle })}
        className="sm:max-w-3xl"
      >
        <DataCreateForm closeSheet={() => setOpenSheet(false)} type={sourceType} />
      </SandwichSheet>
      {!isShareFile && (
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
      )}
    </div>
  )

  return (
    <DataList
      items={sourceType === 'model' && !organization ? organizationItems : resourceItems}
      title={organization || sourceTitle}
      description={
        organization
          ? `${organization} 组织下的模型，共 ${filteredData.length} 个。`
          : sourceType === 'model'
            ? `按组织聚合展示模型，共 ${organizationItems.length} 个组织。`
            : undefined
      }
      showOwner={sourceType !== 'model' || !!organization}
      showDescriptionFallback={sourceType !== 'model' || !!organization}
      compactMetadata={sourceType === 'model' && !organization}
      mainArea={(item) => {
        const repositorySource = item.tag.find(
          (tag) => tag === 'huggingface' || tag === 'modelscope'
        )
        if (sourceType === 'model' && !organization) {
          return (
            <Link
              to="/portal/data/models"
              search={{ organization: item.organization || item.name }}
              className="flex min-w-0 items-center gap-2"
            >
              <RepositorySourceMark
                source={item.source}
                organization={item.organization || item.name}
                logoURL={item.organizationUrl}
                category="model"
                className="size-8"
              />
              <div className="min-w-0">
                <p className="truncate font-mono text-base font-semibold">{item.name}</p>
                <p className="text-muted-foreground text-xs">{item.desc}</p>
              </div>
            </Link>
          )
        }
        return (
          <div className="flex min-w-0 items-center gap-2">
            <RepositorySourceMark
              source={repositorySource}
              organization={item.organization || item.name.split('/')[0]}
              logoURL={item.organizationUrl}
              category={sourceType || 'dataset'}
            />
            <TooltipLink
              {...getLinkOptions(sourceType || 'dataset')}
              params={{ id: `${item.id}` }}
              name={
                <p className="max-w-full truncate text-left font-mono text-[15px] font-semibold sm:max-w-[400px]">
                  {item.name}
                </p>
              }
              tooltip={`查看${sourceTitle}详情`}
              className="min-w-0"
            />
          </div>
        )
      }}
      actionArea={actionArea}
    />
  )
}
