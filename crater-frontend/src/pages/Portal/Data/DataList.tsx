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

import { useMemo, useState } from 'react'
import { Input } from '@/components/ui/input'
//import { Link } from "react-router-dom";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  ArrowDownAZIcon,
  ArrowDownZAIcon,
  BotIcon,
  DatabaseZapIcon,
  EllipsisVerticalIcon,
  PackageIcon,
  SearchIcon,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'
//import TooltipButton from "@/components/custom/TooltipButton";
import TooltipLink from '@/components/label/TooltipLink'
import PageTitle from '@/components/layout/PageTitle'
import TipBadge from '@/components/badge/TipBadge'
import { TimeDistance } from '@/components/custom/TimeDistance'
import { motion } from 'framer-motion'
import { globalUserInfo } from '@/utils/store'
import { useAtomValue } from 'jotai'
// 导入所需组件
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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
import { Trash2Icon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { IUserInfo } from '@/services/api/vcjob'
import UserLabel from '@/components/label/UserLabel'
import Nothing from '@/components/placeholder/Nothing'

export interface DataItem {
  id: number
  name: string
  desc: string
  createdAt?: string
  tag: string[]
  url?: string
  template?: string
  owner: IUserInfo
}
//假设 JobType 是这样定义的枚举
enum JobType {
  Jupyter = 'jupyter',
  Custom = 'custom',
  Tensorflow = 'tensorflow',
  Pytorch = 'pytorch',
}

const getNewJobUrl = (jobType: JobType) => {
  switch (jobType) {
    case JobType.Jupyter: // 直接匹配枚举值
      return 'job/inter/new-jupyter-vcjobs'
    case JobType.Custom:
      return 'job/batch/new-vcjobs'
    case JobType.Tensorflow:
      return 'job/batch/new-tensorflow'
    case JobType.Pytorch:
      return 'job/batch/new-pytorch'
    default:
      return 'job/batch/new-vcjobs'
  }
}

// 新增 JSON 解析函数
const getJobUrlFromTemplate = (template: string): string => {
  try {
    const parsed = JSON.parse(template)

    // 类型安全校验
    if (!parsed.type || !Object.values(JobType).includes(parsed.type)) {
      return getNewJobUrl(JobType.Jupyter)
    }

    // 通过类型断言确保类型安全
    const jobType = parsed.type as JobType
    return getNewJobUrl(jobType)
  } catch {
    return getNewJobUrl(JobType.Jupyter) // 解析失败返回默认
  }
}

export default function DataList({
  items,
  title,
  actionArea,
  handleDelete,
}: {
  items: DataItem[]
  title: string
  actionArea?: React.ReactNode
  handleDelete?: (id: number) => void
}) {
  const [sort, setSort] = useState('ascending')
  const [modelType, setModelType] = useState('所有标签')
  const [searchTerm, setSearchTerm] = useState('')
  const [ownerFilter, setOwnerFilter] = useState('所有') // 修改默认值为"所有"
  const user = useAtomValue(globalUserInfo)

  const tags = useMemo(() => {
    const tags = new Set<string>()
    items.forEach((model) => {
      model.tag.forEach((tag) => tags.add(tag))
    })
    return Array.from(tags)
  }, [items])

  const filteredItems = items
    .sort((a, b) =>
      sort === 'descending'
        ? new Date(a.createdAt || '').getTime() - new Date(b.createdAt || '').getTime()
        : new Date(b.createdAt || '').getTime() - new Date(a.createdAt || '').getTime()
    )
    .filter((item) =>
      modelType === '所有标签' ? true : item.tag.includes(modelType) ? true : false
    )
    .filter((item) => item.name.toLowerCase().includes(searchTerm.toLowerCase()))
    // 修改：基于所有者筛选，添加"所有"选项
    .filter((item) =>
      ownerFilter === '所有'
        ? true
        : ownerFilter === '我的'
          ? user.name === item.owner.username
          : user.name !== item.owner.username
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
                <span>升序</span>
              </div>
            </SelectItem>
            <SelectItem value="descending">
              <div className="flex items-center gap-4">
                <ArrowDownZAIcon size={16} />
                <span>降序</span>
              </div>
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <Separator />
      {filteredItems.length === 0 ? (
        <Nothing />
      ) : (
        <ul className="faded-bottom no-scrollbar grid gap-4 overflow-auto pt-4 pb-16 md:grid-cols-2 lg:grid-cols-3">
          {filteredItems.map((item, index) => (
            <motion.li
              key={item.name}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: (index / 3) * 0.1 }}
              whileHover={{ y: -5 }}
              className="bg-card flex flex-col justify-between gap-3 rounded-lg border hover:shadow-md"
            >
              <div className="flex flex-row items-center justify-between p-4 pb-0">
                <div className="flex items-center gap-2">
                  <div
                    className={`bg-primary/10 text-primary flex size-10 items-center justify-center rounded-lg p-1`}
                  >
                    {title === '模型' ? (
                      <BotIcon />
                    ) : title === '数据集' ? (
                      <DatabaseZapIcon />
                    ) : (
                      <PackageIcon />
                    )}
                  </div>
                  {title === '作业模板' ? (
                    <TooltipLink
                      to={`/portal/${getJobUrlFromTemplate(item.template || '')}?fromTemplate=${item.id}`}
                      name={<p className="text-left">{item.name}</p>}
                      tooltip={`查看${title}详情`}
                      className="font-semibold"
                    />
                  ) : (
                    <TooltipLink
                      to={`${item.id}`}
                      name={<p className="text-left">{item.name}</p>}
                      tooltip={`查看${title}详情`}
                      className="font-semibold"
                    />
                  )}
                </div>
                {user.name === item.owner.username && (
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
                <div className="flex flex-row flex-wrap gap-1 p-4 pt-0">
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
              </div>
            </motion.li>
          ))}
        </ul>
      )}
    </div>
  )
}
