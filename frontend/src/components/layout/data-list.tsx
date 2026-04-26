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
  EllipsisVerticalIcon,
  SearchIcon,
} from 'lucide-react'
import { Trash2Icon } from 'lucide-react'
import { motion } from 'motion/react'
import { ReactNode, useMemo, useState } from 'react'
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
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import TipBadge from '@/components/badge/tip-badge'
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

import { atomUserInfo } from '@/utils/store'

export interface DataItem {
  id: number
  name: string
  desc: string
  createdAt?: string
  mountCount?: number
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
  const [sortField, setSortField] = useState<'createdAt' | 'mountCount'>('mountCount')
  const [modelType, setModelType] = useState('所有标签')
  const [searchTerm, setSearchTerm] = useState('')
  const [ownerFilter, setOwnerFilter] = useState('所有') // 修改默认值为"所有"
  const user = useAtomValue(atomUserInfo)

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

  const filteredItems = [...items]
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
    )

  return (
    <div>
      <PageTitle
        title={title}
        description={`我们为您准备了一些常见${title}，也欢迎您上传并分享更多${title}。`}
      >
        {actionArea}
      </PageTitle>
      <div className="my-4 flex items-end justify-between sm:my-0 sm:items-center">
        <div className="flex flex-col gap-4 sm:my-4 sm:flex-row">
          <div className="relative ml-auto h-9 flex-1 md:grow-0">
            <SearchIcon className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
            <Input
              placeholder={`搜索${title}...`}
              className="h-9 w-40 pl-8 lg:w-[250px]"
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
            onValueChange={(value) => setSortField(value as 'createdAt' | 'mountCount')}
          >
            <SelectTrigger className="min-w-28">
              <SelectValue />
            </SelectTrigger>
            <SelectContent align="end">
              <SelectItem value="createdAt">{t('dataList.sortField.createdAt')}</SelectItem>
              <SelectItem value="mountCount">{t('dataList.sortField.mountCount')}</SelectItem>
            </SelectContent>
          </Select>
          <Select value={sort} onValueChange={setSort}>
            <SelectTrigger className="w-16">
              <SelectValue>
                {sort === 'ascending' ? <ArrowDownAZIcon size={16} /> : <ArrowDownZAIcon size={16} />}
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
        <ul className="faded-bottom no-scrollbar grid gap-4 overflow-auto pt-4 pb-16 md:grid-cols-2 lg:grid-cols-3">
          {filteredItems.map((item, index) => (
            <motion.li
              key={item.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: (index / 3) * 0.1 }}
              whileHover={{ y: -5 }}
              className="bg-card flex flex-col justify-between gap-3 rounded-lg border hover:shadow-md"
            >
              <div className="flex flex-row items-center justify-between p-4 pb-0">
                {mainArea ? <>{mainArea(item)}</> : <></>}
                {user?.name === item.owner.username && (
                  <AlertDialog>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8 p-0">
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

              {item.tag.length > 0 && (
                <div className="flex flex-row flex-wrap gap-1 px-4 pb-1">
                  {item.tag.map((tag) => (
                    <Badge variant="secondary" key={tag} className="rounded-full">
                      {tag}
                    </Badge>
                  ))}
                </div>
              )}
              <p
                className="text-muted-foreground line-clamp-3 px-4 text-sm text-balance"
                title={item.desc}
              >
                {item.desc}
              </p>
              <div>
                <div className="flex items-end justify-between gap-2 p-4 pt-0">
                  <div className="flex flex-row flex-wrap gap-1">
                    <TipBadge
                      title={
                        <UserLabel
                          info={item.owner}
                          className="hover:text-highlight-orange text-xs"
                        />
                      }
                    />
                    <TipBadge
                      title={<TimeDistance date={item.createdAt || '2023'} />}
                      className="bg-purple-600/15 text-purple-600 hover:bg-purple-600/25"
                    />
                  </div>
                  {item.mountCount !== undefined && (
                    <TooltipProvider delayDuration={100}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="text-muted-foreground hover:text-foreground inline-flex cursor-help items-center gap-1 text-xs font-medium">
                            <BarChart3Icon className="size-4" />
                            <span>{item.mountCount}</span>
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>{t('dataList.mountCountTooltip')}</p>
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  )}
                </div>
              </div>
            </motion.li>
          ))}
        </ul>
      )}
    </div>
  )
}
