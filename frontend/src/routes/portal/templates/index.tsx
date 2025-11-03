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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { PackageIcon } from 'lucide-react'

import DocsButton from '@/components/button/docs-button'
import { getNewJobLink } from '@/components/job/new-job-button'
import TooltipLink from '@/components/label/tooltip-link'
import DataList from '@/components/layout/data-list'

import { deleteJobTemplate, listJobTemplate } from '@/services/api/jobtemplate'
import { JobType } from '@/services/api/vcjob'

export const Route = createFileRoute('/portal/templates/')({
  component: RouteComponent,
})

// 新增 JSON 解析函数
const getJobUrlFromTemplate = (template: string) => {
  try {
    const parsed = JSON.parse(template)

    // 类型安全校验
    if (!parsed.type || !Object.values(JobType).includes(parsed.type)) {
      return getNewJobLink(JobType.Jupyter)
    }

    // 通过类型断言确保类型安全
    const jobType = parsed.type as JobType
    return getNewJobLink(jobType)
  } catch {
    return getNewJobLink(JobType.Jupyter) // 解析失败返回默认
  }
}

function RouteComponent() {
  const queryClient = useQueryClient()

  const { data: templateData } = useQuery({
    queryKey: ['data', 'jobtemplate'],
    queryFn: () => listJobTemplate(),
    select: (res) => res.data,
  })

  const { mutate: handleDelete } = useMutation({
    mutationFn: (id: number) => deleteJobTemplate(id),
    onSuccess: () => {
      // 删除成功后，刷新数据
      queryClient.invalidateQueries({ queryKey: ['data', 'jobtemplate'] })
    },
  })

  return (
    <DataList
      items={
        templateData?.map((jobTemplate) => ({
          id: jobTemplate.id,
          name: jobTemplate.name,
          desc: jobTemplate.describe,
          tag: [], // 假设 API 返回的是 tags 字段
          createdAt: jobTemplate.createdAt,
          template: jobTemplate.template,
          owner: jobTemplate.userInfo,
        })) || []
      }
      title="作业模板"
      handleDelete={handleDelete}
      mainArea={(item) => {
        return (
          <div className="flex items-center gap-2">
            <div
              className={`bg-primary/10 text-primary flex size-10 items-center justify-center rounded-lg p-1`}
            >
              <PackageIcon />
            </div>
            <TooltipLink
              {...getJobUrlFromTemplate(item.template || '')}
              search={{ fromTemplate: item.id }}
              name={<p className="text-left">{item.name}</p>}
              tooltip={`使用该模板`}
              className="font-semibold"
            />
          </div>
        )
      }}
      actionArea={
        <div className="flex flex-row gap-3">
          <DocsButton title={'作业模板文档'} url="quick-start/jobtemplate" />
        </div>
      }
    />
  )
}
