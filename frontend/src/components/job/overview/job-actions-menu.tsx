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
import { Link } from '@tanstack/react-router'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { ClockIcon, InfoIcon, RedoDotIcon, SquareIcon, Trash2Icon, XIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

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

import { createApprovalOrder } from '@/services/api/approvalorder'
import { IJobInfo, JobStatus, getJobStateType } from '@/services/api/vcjob'

interface JobActionsMenuProps {
  jobInfo: IJobInfo
  onDelete: (jobName: string) => void
}

export const JobActionsMenu = ({ jobInfo, onDelete }: JobActionsMenuProps) => {
  const jobStatus = getJobStateType(jobInfo.status)
  const option = useMemo(() => {
    return getNewJobLink(jobInfo.jobType)
  }, [jobInfo.jobType])

  const [extensionDialogOpen, setExtensionDialogOpen] = useState(false)
  const [extensionHours, setExtensionHours] = useState('')
  const [reason, setReason] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleExtensionSubmit = async () => {
    if (!extensionHours || !reason.trim()) {
      return
    }

    const hours = parseInt(extensionHours)
    if (isNaN(hours) || hours < 1 || hours > 96) {
      toast.error('延时时长必须在1-96小时之间')
      return
    }

    setIsSubmitting(true)
    try {
      await createApprovalOrder({
        name: `${jobInfo.jobName}`,
        type: 'job',
        status: 'Pending',
        approvalorderTypeID: 1,
        approvalorderReason: reason.trim(),
        approvalorderExtensionHours: hours,
      })

      setExtensionDialogOpen(false)
      setExtensionHours('')
      setReason('')
      toast.success('创建延时申请成功')
    } catch (error) {
      toast.error('创建延时申请失败:' + (error instanceof Error ? error.message : '未知错误'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <AlertDialog>
      <Dialog open={extensionDialogOpen} onOpenChange={setExtensionDialogOpen}>
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
              <Link to="/portal/jobs/detail/$name" params={{ name: jobInfo.jobName }}>
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
            <DropdownMenuItem asChild>
              <DialogTrigger asChild>
                <button className="flex w-full items-center gap-2 px-2 py-1.5 text-sm">
                  <ClockIcon className="text-highlight-blue size-4" />
                  申请延时
                </button>
              </DialogTrigger>
            </DropdownMenuItem>
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

        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>申请作业延时</DialogTitle>
            <DialogDescription>
              为作业 "{jobInfo.jobName}" 申请延时，需要管理员审批。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="extensionHours" className="text-right">
                延时时长
              </Label>
              <div className="col-span-3 flex items-center gap-2">
                <Input
                  id="extensionHours"
                  type="number"
                  min="1"
                  max="96"
                  step="1"
                  placeholder="请输入小时数(1-96)"
                  value={extensionHours}
                  onChange={(e) => setExtensionHours(e.target.value)}
                />
                <span className="text-muted-foreground text-sm">小时</span>
              </div>
            </div>
            <div className="grid grid-cols-4 items-start gap-4">
              <Label htmlFor="reason" className="pt-2 text-right">
                申请原因
              </Label>
              <Textarea
                id="reason"
                className="col-span-3"
                placeholder="请说明申请延时的原因(必填)..."
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setExtensionDialogOpen(false)}
              disabled={isSubmitting}
            >
              取消
            </Button>
            <Button
              onClick={handleExtensionSubmit}
              disabled={!extensionHours || !reason.trim() || isSubmitting}
            >
              {isSubmitting ? '提交中...' : '提交申请'}
            </Button>
          </DialogFooter>
        </DialogContent>

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
      </Dialog>
    </AlertDialog>
  )
}
