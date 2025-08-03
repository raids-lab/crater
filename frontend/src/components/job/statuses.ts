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
import { jobPhases } from '@/components/badge/JobPhaseBadge'
import { jobTypes } from '@/components/badge/JobTypeBadge'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

export const getHeader = (key: string): string => {
  switch (key) {
    case 'id':
      return '序号'
    case 'name':
      return '名称'
    case 'jobType':
      return '类型'
    case 'owner':
      return '用户'
    case 'status':
      return '状态'
    case 'nodes':
      return '节点'
    case 'resources':
      return '资源'
    case 'priority':
      return '优先级'
    case 'profileStatus':
      return '分析状态'
    case 'createdAt':
      return '创建于'
    case 'startedAt':
      return '开始于'
    case 'completedAt':
      return '结束于'
    default:
      return key
  }
}

export const jobToolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: '搜索作业名称',
    key: 'name',
  },
  filterOptions: [
    {
      key: 'jobType',
      title: '类型',
      option: jobTypes,
    },
    {
      key: 'status',
      title: '状态',
      option: jobPhases,
    },
  ],
  getHeader: getHeader,
}
