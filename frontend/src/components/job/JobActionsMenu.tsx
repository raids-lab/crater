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

import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
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
import { DotsHorizontalIcon } from '@radix-ui/react-icons'
import { InfoIcon, RedoDotIcon, SquareIcon, Trash2Icon, XIcon } from 'lucide-react'
import { getJobStateType, IJobInfo, JobStatus } from '@/services/api/vcjob'
import { useMemo } from 'react'
import { getNewJobUrl } from '@/utils/job'

interface JobActionsMenuProps {
  jobInfo: IJobInfo
  onDelete: (jobName: string) => void
}

export const JobActionsMenu = ({ jobInfo, onDelete }: JobActionsMenuProps) => {
  const jobStatus = getJobStateType(jobInfo.status)
  const newJobUrl = useMemo(() => getNewJobUrl(jobInfo.jobType), [jobInfo.jobType])

  return (
    <AlertDialog>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="h-8 w-8 p-0">
            <span className="sr-only">操作</span>
            <DotsHorizontalIcon className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel className="text-muted-foreground text-xs">操作</DropdownMenuLabel>
          <Link to={`${jobInfo.jobName}`}>
            <DropdownMenuItem>
              <InfoIcon className="text-highlight-emerald size-4" />
              详情
            </DropdownMenuItem>
          </Link>
          <Link to={`${newJobUrl}?fromJob=${jobInfo.jobName}`}>
            <DropdownMenuItem>
              <RedoDotIcon className="text-highlight-purple size-4" />
              克隆
            </DropdownMenuItem>
          </Link>
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
    </AlertDialog>
  )
}
