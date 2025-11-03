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
import { Link, linkOptions } from '@tanstack/react-router'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { ClockIcon, InfoIcon, RedoDotIcon, SquareIcon, Trash2Icon, XIcon } from 'lucide-react'
import { useMemo } from 'react'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import ExtensionRequestDialog from '@/components/job/extension-request-dialog'
import { getNewJobLink } from '@/components/job/new-job-button'
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

import { IJobInfo, JobStatus, getJobStateType } from '@/services/api/vcjob'

// Link Options for portal job navigation
const portalJobDetailLinkOptions = linkOptions({
  to: '/portal/jobs/detail/$name',
  params: { name: '' },
  search: { tab: '' },
})

interface JobActionsMenuProps {
  jobInfo: IJobInfo
  onDelete: (jobName: string) => void
}

export const JobActionsMenu = ({ jobInfo, onDelete }: JobActionsMenuProps) => {
  const jobStatus = getJobStateType(jobInfo.status)
  const option = useMemo(() => {
    return getNewJobLink(jobInfo.jobType)
  }, [jobInfo.jobType])

  // 暂时隐藏申请锁定功能，工单审批功能不完善
  const canExtend = true // jobStatus === JobStatus.Running

  // extension request logic moved to ExtensionRequestDialog

  return (
    <AlertDialog>
      {/* Dialog wrapper removed; ExtensionRequestDialog manages its own dialog state */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="h-8 w-8 p-0">
            <span className="sr-only">操作</span>
            <DotsHorizontalIcon className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel className="text-muted-foreground text-xs">操作</DropdownMenuLabel>
          <DropdownMenuItem asChild>
            <Link {...portalJobDetailLinkOptions} params={{ name: jobInfo.jobName }}>
              <InfoIcon className="text-highlight-emerald size-4" />
              详情
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem asChild>
            <Link {...option} search={{ fromJob: jobInfo.jobName }}>
              <RedoDotIcon className="text-highlight-purple size-4" />
              克隆
            </Link>
          </DropdownMenuItem>
          {canExtend && jobStatus === JobStatus.Running && (
            <DropdownMenuItem asChild>
              <ExtensionRequestDialog
                jobName={jobInfo.jobName}
                trigger={
                  <button className="flex w-full items-center gap-2 px-2 py-1.5 text-sm">
                    <ClockIcon className="text-highlight-blue size-4" />
                    申请锁定
                  </button>
                }
              />
            </DropdownMenuItem>
          )}
          <AlertDialogTrigger asChild>
            <DropdownMenuItem className="group">
              {jobStatus === JobStatus.NotStarted ? (
                <XIcon className="text-highlight-orange size-4" />
              ) : jobStatus === JobStatus.Running ? (
                <SquareIcon className="text-highlight-orange size-4" />
              ) : (
                <Trash2Icon className="text-destructive size-4" />
              )}
              {jobStatus === JobStatus.NotStarted
                ? '取消'
                : jobStatus === JobStatus.Running
                  ? '停止'
                  : '删除'}
            </DropdownMenuItem>
          </AlertDialogTrigger>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* ExtensionRequestDialog 替代内联 DialogContent */}

      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {jobStatus === JobStatus.NotStarted
              ? '取消作业'
              : jobStatus === JobStatus.Running
                ? '停止作业'
                : '删除作业'}
          </AlertDialogTitle>
          <AlertDialogDescription>
            作业 {jobInfo.name} 将
            {jobStatus === JobStatus.NotStarted
              ? '取消，是否放弃排队？'
              : jobStatus === JobStatus.Running
                ? '停止，请确认已经保存好所需数据。'
                : '删除，所有数据将被清理。'}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={() => onDelete(jobInfo.jobName)}>
            {jobStatus === JobStatus.NotStarted
              ? '确认'
              : jobStatus === JobStatus.Running
                ? '停止'
                : '删除'}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
      {/* urgent dialog handled inside ExtensionRequestDialog */}
    </AlertDialog>
  )
}
