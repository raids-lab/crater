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
import { useQuery } from '@tanstack/react-query'
import { ClockIcon } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'

import { CopyButton } from '@/components/button/copy-button'
import { DurationFields } from '@/components/form/duration-fields'
import FormLabelMust from '@/components/form/form-label-must'
import { MarkdownRenderer } from '@/components/form/markdown-renderer'
import SandwichSheet, { SandwichLayout } from '@/components/sheet/sandwich-sheet'

import {
  ApprovalOrder,
  createApprovalOrder,
  listMyApprovalOrder,
} from '@/services/api/approvalorder'
import { NodeStatus } from '@/services/api/cluster'
import { IJobInfo } from '@/services/api/vcjob'
import { queryNodes } from '@/services/query/node'

const ExtensionMarkdown = `
## 清理规则
- 如果申请了 GPU 资源，当过去 2 个小时 GPU 利用率为 0，我们将尝试发送告警信息给用户，建议用户检查作业是否正常运行。若此后半小时 GPU 利用率仍为 0，系统将释放作业占用的资源。
- 当作业运行超过 4 天，我们将尝试发送告警信息给用户，提醒用户作业运行时间过长；若此后一天内用户未联系管理员说明情况并锁定作业，系统将释放作业占用的资源。

## 自动审批规则
- 每隔2天，第一个小于12小时的作业锁定工单，系统会直接审批通过
`

interface JobLockSheetProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
  jobName: string
}

export const JobLockSheet = ({ isOpen, onOpenChange, jobName }: JobLockSheetProps) => {
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [urgentOpen, setUrgentOpen] = useState(false)
  const [newlyCreatedOrder, setNewlyCreatedOrder] = useState<ApprovalOrder | null>(null)
  const [duration, setDuration] = useState<{ days: number; hours: number; totalHours: number }>({
    days: 0,
    hours: 0,
    totalHours: 0,
  })
  const [reason, setReason] = useState('')

  const commonReasons = ['模型训练尚未结束', '需要保留环境进行调试', '等待数据处理']

  const handleDurationChange = useCallback(
    (val: { days: number; hours: number; totalHours: number }) => setDuration(val),
    []
  )

  const handleSubmit = async () => {
    const hours = duration.totalHours
    if (!reason || reason.trim().length === 0) {
      toast.error('请填写申请原因')
      return
    }
    if (hours < 1) {
      toast.error('锁定时长必须至少为 1 小时')
      return
    }

    setIsSubmitting(true)
    try {
      const reasonString = reason.trim()

      try {
        const myOrdersResp = await listMyApprovalOrder()
        const duplicates = (myOrdersResp.data || []).filter(
          (o: ApprovalOrder) => o.name === jobName && o.status === 'Pending'
        )
        const found = duplicates.some((o: ApprovalOrder) => {
          const existing = (o.content?.approvalorderReason || '') as string
          return existing === reasonString || existing.includes(reasonString)
        })
        if (found) {
          const ok = window.confirm('检测到已有相似的申请（相同标题或理由），是否仍要继续提交？')
          if (!ok) {
            setIsSubmitting(false)
            return
          }
        }
      } catch (err) {
        // eslint-disable-next-line no-console
        console.warn('重复检测失败', err)
      }

      await createApprovalOrder({
        name: jobName,
        type: 'job',
        status: 'Pending',
        approvalorderTypeID: 1,
        approvalorderReason: reasonString,
        approvalorderExtensionHours: hours,
      })
      onOpenChange(false)
      setDuration({ days: 0, hours: 0, totalHours: 0 })
      setReason('')
      toast.success('创建锁定申请成功')

      const myOrders = await listMyApprovalOrder()
      const latestOrder = myOrders.data
        .filter((order: ApprovalOrder) => order.name === jobName)
        .sort(
          (a: ApprovalOrder, b: ApprovalOrder) =>
            new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
        )[0]

      if (latestOrder && latestOrder.status === 'Pending') {
        setNewlyCreatedOrder(latestOrder)
        setUrgentOpen(true)
      }
    } catch (error) {
      toast.error('创建锁定申请失败:' + (error instanceof Error ? error.message : '未知错误'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
      <SandwichSheet
        isOpen={isOpen}
        onOpenChange={onOpenChange}
        title="申请作业锁定"
        description={`为作业 “${jobName}” 申请锁定，需要管理员审批。`}
        className="w-full sm:w-[25vw] sm:max-w-none sm:min-w-[600px]"
      >
        <SandwichLayout
          footer={
            <>
              <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
                取消
              </Button>
              <Button
                onClick={handleSubmit}
                disabled={isSubmitting || duration.totalHours < 1 || !reason.trim()}
              >
                {isSubmitting ? '提交中...' : '提交申请'}
              </Button>
            </>
          }
        >
          <div className="bg-muted/40 mt-4 rounded-md p-4">
            <MarkdownRenderer>{ExtensionMarkdown}</MarkdownRenderer>
          </div>

          <div className="grid gap-6 py-6">
            <div>
              <div className="mb-2 block text-sm">
                延长时间
                <span className="ml-1 inline-block align-middle">
                  <FormLabelMust />
                </span>
              </div>
              <DurationFields
                value={{ days: duration.days, hours: duration.hours }}
                onChange={handleDurationChange}
                origin={null}
                showPreview={true}
              />
            </div>

            <div>
              <div className="mb-2">
                <div className="text-sm font-medium">
                  申请原因
                  <span className="ml-1 inline-block align-middle">
                    <FormLabelMust />
                  </span>
                </div>
              </div>

              <div>
                <Input
                  value={reason}
                  onChange={(e) => setReason((e.target as HTMLInputElement).value)}
                  className="w-full"
                  placeholder="请输入申请原因"
                />
                <div className="flex gap-2 overflow-x-auto pt-2 pb-1">
                  {commonReasons.map((r) => (
                    <Badge
                      key={r}
                      variant="outline"
                      className="shrink-0 cursor-pointer border-orange-600 bg-orange-50 text-orange-600 transition-colors hover:bg-orange-100 dark:border-orange-500 dark:bg-orange-950 dark:text-orange-400 dark:hover:bg-orange-900"
                      onClick={() => setReason(r)}
                    >
                      {r}
                    </Badge>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </SandwichLayout>
      </SandwichSheet>

      {newlyCreatedOrder && (
        <Dialog open={urgentOpen} onOpenChange={setUrgentOpen}>
          <DialogContent>
            <div className="mb-4">
              <h3 className="text-lg font-medium">紧急审批提醒</h3>
              <p className="text-muted-foreground text-sm">
                您的锁定申请已提交。如需紧急审批，请将以下链接发送给管理员。
              </p>
            </div>
            <div className="bg-muted/40 my-4 flex items-center justify-between space-x-2 rounded-lg p-3">
              <pre className="text-muted-foreground overflow-auto text-sm">
                {`${window.location.origin}/admin/more/orders/${newlyCreatedOrder.id}`}
              </pre>
              <CopyButton
                content={`${window.location.origin}/admin/more/orders/${newlyCreatedOrder.id}`}
              />
            </div>
            <div className="flex justify-end">
              <Button onClick={() => setUrgentOpen(false)}>关闭</Button>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
  )
}

interface JobLockMenuItemProps {
  jobInfo: IJobInfo
  onLock: () => void
}

export const JobLockMenuItem = ({ jobInfo, onLock }: JobLockMenuItemProps) => {
  const { data: nodes } = useQuery(queryNodes())

  const areNodesReady = useMemo(() => {
    if (!jobInfo.nodes || jobInfo.nodes.length === 0) return true
    if (!nodes) return true

    const jobNodes = nodes.filter((node) => jobInfo.nodes.includes(node.name))
    if (jobNodes.length === 0) return true

    return jobNodes.every((node) => node.status === NodeStatus.Ready)
  }, [jobInfo.nodes, nodes])

  const handleLockClick = (e: React.MouseEvent) => {
    if (!areNodesReady) {
      e.preventDefault()
      toast.error('作业所在节点未在正常运行，暂不支持锁定作业')
    } else {
      onLock()
    }
  }

  return (
    <DropdownMenuItem
      onClick={handleLockClick}
      className={!areNodesReady ? 'cursor-not-allowed opacity-50' : ''}
    >
      <div className="flex w-full items-center gap-2">
        <ClockIcon className="text-highlight-blue size-4" />
        申请锁定
      </div>
    </DropdownMenuItem>
  )
}
