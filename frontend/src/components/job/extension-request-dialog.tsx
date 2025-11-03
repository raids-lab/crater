import { useCallback, useState } from 'react'
// 使用本地 state 管理单行原因输入
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
import { Input } from '@/components/ui/input'

import { CopyButton } from '@/components/button/copy-button'
import { DurationFields } from '@/components/form/duration-fields'
import FormLabelMust from '@/components/form/form-label-must'
// TagsInput removed in favor of ReasonInput

import { MarkdownRenderer } from '@/components/form/markdown-renderer'

import {
  ApprovalOrder,
  createApprovalOrder,
  listMyApprovalOrder,
} from '@/services/api/approvalorder'

interface ExtensionRequestDialogProps {
  jobName: string
  trigger?: React.ReactNode
}

const ExtensionMarkdown = `
## 清理规则
- 如果申请了 GPU 资源，当过去 2 个小时 GPU 利用率为 0，我们将尝试发送告警信息给用户，建议用户检查作业是否正常运行。若此后半小时 GPU 利用率仍为 0，系统将释放作业占用的资源。
- 当作业运行超过 4 天，我们将尝试发送告警信息给用户，提醒用户作业运行时间过长；若此后一天内用户未联系管理员说明情况并锁定作业，系统将释放作业占用的资源。

## 自动审批规则
- 每隔2天，第一个小于12小时的作业锁定工单，系统会直接审批通过
`

export default function ExtensionRequestDialog({ jobName, trigger }: ExtensionRequestDialogProps) {
  const [open, setOpen] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [urgentOpen, setUrgentOpen] = useState(false)
  const [newlyCreatedOrder, setNewlyCreatedOrder] = useState<ApprovalOrder | null>(null)
  const [duration, setDuration] = useState<{ days: number; hours: number; totalHours: number }>({
    days: 0,
    hours: 0,
    totalHours: 0,
  })

  // 使用受控 Input 管理单行原因（简化，不用 react-hook-form）
  const [reason, setReason] = useState('')

  const handleDurationChange = useCallback(
    (val: { days: number; hours: number; totalHours: number }) => setDuration(val),
    []
  )

  const canExtend = true

  const handleSubmit = async () => {
    if (!canExtend) {
      toast.error('当前作业不是运行中，无法申请锁定')
      return
    }
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
      // 使用单字符串原因（由单行 Input 提供）
      const reasonString = reason.trim()

      // 重复检测：检查当前用户对该 job 是否已有相同标题或相同 reason 的待审批工单
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
        // 如果重复检测失败，不阻止提交，但记录日志
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
      setOpen(false)
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
      <Dialog open={open} onOpenChange={setOpen}>
        {trigger ? (
          <DialogTrigger asChild>{trigger}</DialogTrigger>
        ) : (
          <DialogTrigger asChild>
            <button className="flex w-full items-center gap-2 px-2 py-1.5 text-sm">申请锁定</button>
          </DialogTrigger>
        )}
        <DialogContent className="w-full sm:max-w-[760px] md:max-w-[880px]">
          <DialogHeader>
            <DialogTitle>申请作业锁定</DialogTitle>
            <DialogDescription>为作业 “{jobName}” 申请锁定，需要管理员审批。</DialogDescription>

            <div className="bg-muted/40 mt-4 rounded-md p-4">
              <MarkdownRenderer>{ExtensionMarkdown}</MarkdownRenderer>
            </div>
          </DialogHeader>

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
              {/* 申请原因（垂直布局）：使用 TagsInput 支持预设与自由输入 */}
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
                />
              </div>
            </div>
          </div>

          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setOpen(false)} disabled={isSubmitting}>
              取消
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={isSubmitting || duration.totalHours < 1 || !reason.trim()}
            >
              {isSubmitting ? '提交中...' : '提交申请'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
                {`${window.location.origin}/admin/settings/orders/${newlyCreatedOrder.id}`}
              </pre>
              <CopyButton
                content={`${window.location.origin}/admin/settings/orders/${newlyCreatedOrder.id}`}
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
