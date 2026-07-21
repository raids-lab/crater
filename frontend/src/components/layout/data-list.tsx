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
import { useAtomValue } from 'jotai'
import {
  ArrowDownAZIcon,
  ArrowDownZAIcon,
  BarChart3Icon,
  ClockIcon,
  DownloadIcon,
  EllipsisVerticalIcon,
  Globe2Icon,
  HardDriveIcon,
  HeartIcon,
  SearchIcon,
} from 'lucide-react'
import { Trash2Icon } from 'lucide-react'
import { ReactNode, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import { TimeDistance } from '@/components/custom/time-distance'
import UserLabel from '@/components/label/user-label'
import PageTitle from '@/components/layout/page-title'
import Nothing from '@/components/placeholder/nothing'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'

import { IUserInfo } from '@/services/api/vcjob'

import { formatFileSize } from '@/utils/file-size'
import { atomUserInfo } from '@/utils/store'

export interface DataItem {
  id: number
  name: string
  desc: string
  createdAt?: string
  updatedAt?: string
  sourceUpdatedAt?: string
  mountCount?: number
  sizeBytes?: number
  downloadCount?: number
  likes?: number
  source?: string
  organization?: string
  organizationUrl?: string
  tag: string[]
  url?: string
  template?: string
  owner: IUserInfo
}

export default function DataList({
  items,
  title,
  mainArea,
  actionArea,
  handleDelete,
}: {
  items: DataItem[]
  title: string
  mainArea?: (data: DataItem) => ReactNode
  actionArea?: ReactNode
  handleDelete?: (id: number) => void
}) {
  const { t } = useTranslation()
  const [sort, setSort] = useState('descending')
  const hasMountCount = useMemo(() => items.some((item) => item.mountCount !== undefined), [items])
  const [sortField, setSortField] = useState<'createdAt' | 'mountCount'>(
    hasMountCount ? 'mountCount' : 'createdAt'
  )
  const [sortFieldManuallyChanged, setSortFieldManuallyChanged] = useState(false)
  const [modelType, setModelType] = useState('所有标签')
  const [searchTerm, setSearchTerm] = useState('')
  const [ownerFilter, setOwnerFilter] = useState('所有') // 修改默认值为"所有"
  const user = useAtomValue(atomUserInfo)

  useEffect(() => {
    const nextDefaultSortField = hasMountCount ? 'mountCount' : 'createdAt'

    if (!sortFieldManuallyChanged && sortField !== nextDefaultSortField) {
      setSortField(nextDefaultSortField)
    }

    if (!hasMountCount && sortField === 'mountCount') {
      setSortField('createdAt')
    }
  }, [hasMountCount, sortField, sortFieldManuallyChanged])

  const tags = useMemo(() => {
    const tags = new Set<string>()
    items.forEach((model) => {
      model.tag.forEach((tag) => tags.add(tag))
    })
    return Array.from(tags)
  }, [items])

  const toSortableNumber = (value: unknown): number => {
    if (typeof value === 'number') {
      return Number.isFinite(value) ? value : 0
    }
    const numericValue = Number(value)
    return Number.isFinite(numericValue) ? numericValue : 0
  }

  // Memoize sorting and filtering to keep large resource lists responsive.
  const filteredItems = useMemo(
    () =>
      [...items]
        .sort((a, b) => {
          const direction = sort === 'descending' ? -1 : 1

          if (sortField === 'mountCount') {
            const aCount = toSortableNumber(a.mountCount)
            const bCount = toSortableNumber(b.mountCount)

            if (aCount !== bCount) {
              return (aCount - bCount) * direction
            }
          } else {
            const aTime = toSortableNumber(new Date(a.createdAt || '').getTime())
            const bTime = toSortableNumber(new Date(b.createdAt || '').getTime())

            if (aTime !== bTime) {
              return (aTime - bTime) * direction
            }
          }

          const aCreatedAt = toSortableNumber(new Date(a.createdAt || '').getTime())
          const bCreatedAt = toSortableNumber(new Date(b.createdAt || '').getTime())
          if (aCreatedAt !== bCreatedAt) {
            return (aCreatedAt - bCreatedAt) * direction
          }

          return (a.id - b.id) * direction
        })
        .filter((item) =>
          modelType === '所有标签' ? true : item.tag.includes(modelType) ? true : false
        )
        .filter((item) => item.name.toLowerCase().includes(searchTerm.toLowerCase()))
        // 修改：基于所有者筛选，添加"所有"选项
        .filter((item) =>
          ownerFilter === '所有'
            ? true
            : ownerFilter === '我的'
              ? user?.name === item.owner.username
              : user?.name !== item.owner.username
        ),
    [items, sort, sortField, modelType, searchTerm, ownerFilter, user?.name]
  )

  return (
    <div>
      <PageTitle
        title={title}
        description={`我们为您准备了一些常见${title}，也欢迎您上传并分享更多${title}。`}
      >
        {actionArea}
      </PageTitle>
      <div className="my-4 flex flex-col gap-3 sm:my-0 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex w-full min-w-0 flex-col gap-3 sm:my-4 sm:w-auto sm:flex-row sm:flex-nowrap sm:items-center">
          <div className="relative h-9 w-full min-w-0 sm:ml-auto sm:w-auto sm:flex-none">
            <SearchIcon className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
            <Input
              placeholder={`搜索${title}...`}
              className="h-9 w-full min-w-0 pl-8 sm:w-40 lg:w-[250px]"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          {title !== '作业模板' && (
            <Select value={modelType} onValueChange={setModelType}>
              <SelectTrigger className="min-w-36">
                <SelectValue>{modelType}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="所有标签">所有标签</SelectItem>
                {tags.map((tag) => (
                  <SelectItem key={tag} value={tag}>
                    {tag}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}

          {/* 新增：简化的所有者筛选 */}
          <Select value={ownerFilter} onValueChange={setOwnerFilter}>
            <SelectTrigger className="min-w-28">
              <SelectValue>{ownerFilter}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="所有">所有{title}</SelectItem>
              <SelectItem value="我的">我的{title}</SelectItem>
              <SelectItem value="他人">他人{title}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center gap-2">
          <Select
            value={sortField}
            onValueChange={(value) => {
              setSortFieldManuallyChanged(true)
              setSortField(value as 'createdAt' | 'mountCount')
            }}
          >
            <SelectTrigger className="min-w-28">
              <SelectValue />
            </SelectTrigger>
            <SelectContent align="end">
              <SelectItem value="createdAt">{t('dataList.sortField.createdAt')}</SelectItem>
              {hasMountCount && (
                <SelectItem value="mountCount">{t('dataList.sortField.mountCount')}</SelectItem>
              )}
            </SelectContent>
          </Select>
          <Select value={sort} onValueChange={setSort}>
            <SelectTrigger className="w-16">
              <SelectValue>
                {sort === 'ascending' ? (
                  <ArrowDownAZIcon size={16} />
                ) : (
                  <ArrowDownZAIcon size={16} />
                )}
              </SelectValue>
            </SelectTrigger>
            <SelectContent align="end">
              <SelectItem value="ascending">
                <div className="flex items-center gap-4">
                  <ArrowDownAZIcon size={16} />
                  <span>{t('dataList.sortDirection.ascending')}</span>
                </div>
              </SelectItem>
              <SelectItem value="descending">
                <div className="flex items-center gap-4">
                  <ArrowDownZAIcon size={16} />
                  <span>{t('dataList.sortDirection.descending')}</span>
                </div>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
      <Separator />
      {filteredItems.length === 0 ? (
        <Nothing />
      ) : (
        <ul className="faded-bottom no-scrollbar grid min-w-0 gap-3 overflow-auto pt-4 pb-16 md:grid-cols-2">
          {filteredItems.map((item, index) => (
            // Keep entry animation CSS-only and cap the stagger so large lists
            // do not accumulate long JavaScript animation delays.
            <li
              key={item.id}
              style={{ animationDelay: `${Math.min(index, 12) * 40}ms` }}
              className="group bg-card hover:border-primary/35 animate-in fade-in-0 slide-in-from-bottom-2 fill-mode-backwards flex min-w-0 flex-col gap-1.5 rounded-lg border px-3 py-2.5 shadow-sm transition-all duration-200 hover:shadow-md"
            >
              <div className="flex min-w-0 flex-row items-center justify-between gap-2">
                {mainArea ? <>{mainArea(item)}</> : <></>}
                {user?.name === item.owner.username && (
                  <AlertDialog>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="text-muted-foreground hover:text-foreground h-8 w-8 shrink-0 p-0 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
                        >
                          <span className="sr-only">更多操作</span>
                          <EllipsisVerticalIcon className="size-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel className="text-muted-foreground text-xs">
                          操作
                        </DropdownMenuLabel>
                        {handleDelete && (
                          <AlertDialogTrigger asChild>
                            <DropdownMenuItem className="group">
                              <Trash2Icon className="text-destructive mr-2 size-4" />
                              删除
                            </DropdownMenuItem>
                          </AlertDialogTrigger>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>删除{title}</AlertDialogTitle>
                        <AlertDialogDescription>
                          {title} {item.name} 将被删除，此操作不可恢复。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDelete?.(item.id)
                          }}
                        >
                          删除
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                )}
              </div>

              <div className="text-muted-foreground flex min-w-0 items-center gap-1.5 overflow-hidden text-xs">
                {item.tag.map((tag) => (
                  <Badge
                    variant="secondary"
                    key={tag}
                    className="h-5 shrink-0 rounded-md px-1.5 text-[10px] font-normal"
                  >
                    {tag}
                  </Badge>
                ))}
                {item.desc && item.tag.length === 0 && (
                  <>
                    <span aria-hidden="true">·</span>
                    <span className="truncate" title={item.desc}>
                      {item.desc}
                    </span>
                  </>
                )}
              </div>
              <div className="text-muted-foreground flex min-w-0 items-center gap-x-3 overflow-hidden text-xs">
                {item.sizeBytes !== undefined && item.sizeBytes > 0 && (
                  <span className="inline-flex shrink-0 items-center gap-1">
                    <HardDriveIcon className="size-3.5" />
                    {formatFileSize(item.sizeBytes)}
                  </span>
                )}
                {item.sourceUpdatedAt && (
                  <span
                    className="inline-flex shrink-0 items-center gap-1"
                    title={t('dataList.sourceUpdatedAt')}
                  >
                    <Globe2Icon className="size-3.5" />
                    <TimeDistance date={item.sourceUpdatedAt} />
                  </span>
                )}
                <span
                  className="inline-flex shrink-0 items-center gap-1"
                  title={t('dataList.craterUpdatedAt')}
                >
                  <ClockIcon className="size-3.5" />
                  <TimeDistance date={item.updatedAt || item.createdAt || '2023'} />
                </span>
                {item.downloadCount !== undefined && item.downloadCount > 0 && (
                  <span className="inline-flex shrink-0 items-center gap-1">
                    <DownloadIcon className="size-3.5" />
                    {item.downloadCount.toLocaleString()}
                  </span>
                )}
                {item.likes !== undefined && item.likes > 0 && (
                  <span className="inline-flex shrink-0 items-center gap-1">
                    <HeartIcon className="size-3.5" />
                    {item.likes.toLocaleString()}
                  </span>
                )}
                <UserLabel
                  info={item.owner}
                  className="hover:text-highlight-orange min-w-0 truncate text-xs"
                />
                {item.mountCount !== undefined && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        aria-label={t('dataList.mountCount', { count: item.mountCount })}
                        className="text-muted-foreground hover:text-foreground inline-flex cursor-help items-center gap-1 text-xs font-medium"
                      >
                        <BarChart3Icon className="size-4" aria-hidden="true" />
                        <span>{item.mountCount}</span>
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{t('dataList.mountCountTooltip')}</p>
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
