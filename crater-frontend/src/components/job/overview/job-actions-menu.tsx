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
import { useCallback, useMemo, useState } from 'react'
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
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { DurationFields } from '@/components/form/DurationFields'
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
  const [reason, setReason] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const [duration, setDuration] = useState<{ days: number; hours: number; totalHours: number }>({
    days: 0,
    hours: 0,
    totalHours: 0,
  })

  const handleDurationChange = useCallback(
    (val: { days: number; hours: number; totalHours: number }) => setDuration(val),
    []
  )

  const canExtend = jobStatus === JobStatus.Running

  const handleExtensionSubmit = async () => {
    if (!canExtend) {
      toast.error('当前作业不是运行中，无法申请延时')
      return
    }
    const hours = duration.totalHours
    if (hours < 1 || !reason.trim()) {
      toast.error('延时时长必须至少为 1 小时，并填写申请原因')
      return
    }

    setIsSubmitting(true)
    try {
      await createApprovalOrder({
        name: jobInfo.jobName,
        type: 'job',
        status: 'Pending',
        approvalorderTypeID: 1,
        approvalorderReason: reason.trim(),
        approvalorderExtensionHours: hours,
      })
      setExtensionDialogOpen(false)
      setDuration({ days: 0, hours: 0, totalHours: 0 })
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
            {canExtend && (
              <DropdownMenuItem asChild>
                <DialogTrigger asChild>
                  <button className="flex w-full items-center gap-2 px-2 py-1.5 text-sm">
                    <ClockIcon className="text-highlight-blue size-4" />
                    申请延时
                  </button>
                </DialogTrigger>
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

        {canExtend && (
          <DialogContent className="w-full sm:max-w-[760px] md:max-w-[880px]">
            <DialogHeader>
              <DialogTitle>申请作业延时</DialogTitle>
              <DialogDescription>
                为作业 “{jobInfo.jobName}” 申请延时，需要管理员审批。
              </DialogDescription>

              <div className="bg-muted/40 mt-4 rounded-md p-4">
                <div className="text-foreground mb-2 text-sm font-semibold">清理规则</div>
                <ul className="text-muted-foreground list-disc space-y-1 pl-5 text-xs leading-relaxed">
                  <li>
                    如果申请了 GPU 资源，当过去 2 个小时 GPU 利用率为
                    0，我们将尝试发送告警信息给用户，建议用户检查作业是否正常运行。若此后半小时 GPU
                    利用率仍为 0，系统将释放作业占用的资源。
                  </li>
                  <li>
                    当作业运行超过 4
                    天，我们将尝试发送告警信息给用户，提醒用户作业运行时间过长；若此后一天内用户未联系管理员说明情况并锁定作业，系统将释放作业占用的资源。
                  </li>
                </ul>
              </div>
            </DialogHeader>

            <div className="grid gap-6 py-6">
              <div>
                <Label className="mb-2 block text-sm">延长时间</Label>
                <DurationFields
                  value={{ days: duration.days, hours: duration.hours }}
                  onChange={handleDurationChange}
                  origin={null}
                  showPreview={true}
                />
              </div>

              <div className="grid grid-cols-4 items-start gap-4">
                <Label htmlFor="reason" className="pt-2 text-right">
                  申请原因
                </Label>
                <Textarea
                  id="reason"
                  className="col-span-3 min-h-[96px] resize-y"
                  placeholder="请说明申请延时的原因(必填)..."
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  rows={4}
                />
              </div>
            </div>

            <DialogFooter className="gap-2">
              <Button
                variant="outline"
                onClick={() => setExtensionDialogOpen(false)}
                disabled={isSubmitting}
              >
                取消
              </Button>
              <Button
                onClick={handleExtensionSubmit}
                disabled={isSubmitting || duration.totalHours < 1 || !reason.trim()}
              >
                {isSubmitting ? '提交中...' : '提交申请'}
              </Button>
            </DialogFooter>
          </DialogContent>
        )}

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
