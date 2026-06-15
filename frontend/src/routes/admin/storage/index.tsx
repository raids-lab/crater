import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { BotMessageSquare, Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'

import {
  LLMDecisionResponse,
  PagedUserSpaces,
  UserSpace,
  apiAdminApplyExpansion,
  apiAdminFreezeJobs,
  apiAdminGetLLMDecisionStatus,
  apiAdminGetUserSpaces,
  apiAdminRevertExpansion,
  apiAdminSetUserSpaceQuota,
  apiAdminStartLLMDecision,
  apiAdminUnfreezeJobs,
} from '@/services/api/storage'
import { IResponse } from '@/services/types'

import StorageDirectoryComparePanel from './-components/storage-directory-compare-panel'
import StorageGovernancePanel from './-components/storage-governance-panel'
import StorageIndexPanel from './-components/storage-index-panel'

export const Route = createFileRoute('/admin/storage/')({
  component: StorageManagementPage,
})

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null) {
    const candidate = error as {
      data?: { msg?: string }
      response?: { data?: { msg?: string } }
    }
    return candidate.data?.msg ?? candidate.response?.data?.msg ?? fallback
  }
  return fallback
}

function shrinkStageLabel(stage?: string): string {
  switch (stage) {
    case 'expanded':
      return '扩容阶段'
    case 'buffer_reduction':
      return '缩容缓冲期'
    default:
      return '无'
  }
}

const QUOTA_UNIT_BYTES = {
  B: 1,
  KB: 1024,
  MB: 1024 ** 2,
  GB: 1024 ** 3,
  TB: 1024 ** 4,
} as const

function roundQuotaValue(value: number): number {
  if (value >= 100) return Math.round(value)
  if (value >= 10) return Number(value.toFixed(1))
  return Number(value.toFixed(2))
}

function normalizeQuotaDisplay(value: number, unit: string): { value: number; unit: string } {
  if (!Number.isFinite(value) || value < 0 || unit === 'unlimited') {
    return { value: Math.max(0, value || 0), unit }
  }

  const unitBytes = QUOTA_UNIT_BYTES[unit as keyof typeof QUOTA_UNIT_BYTES]
  if (!unitBytes) return { value, unit }

  const bytes = Math.round(value * unitBytes)
  const orderedUnits: Array<keyof typeof QUOTA_UNIT_BYTES> = ['TB', 'GB', 'MB', 'KB', 'B']

  const exactUnit = orderedUnits.find((candidate) => {
    const candidateBytes = QUOTA_UNIT_BYTES[candidate]
    return bytes >= candidateBytes && bytes % candidateBytes === 0
  })
  if (exactUnit) {
    return {
      value: bytes / QUOTA_UNIT_BYTES[exactUnit],
      unit: exactUnit,
    }
  }

  const fallbackUnit = orderedUnits.find((candidate) => bytes >= QUOTA_UNIT_BYTES[candidate]) ?? 'B'
  return {
    value: roundQuotaValue(bytes / QUOTA_UNIT_BYTES[fallbackUnit]),
    unit: fallbackUnit,
  }
}

export default function StorageManagementPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  // 设置配额弹窗
  const [isQuotaDialogOpen, setIsQuotaDialogOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<UserSpace | null>(null)
  const [quotaValue, setQuotaValue] = useState<number>(0)
  const [quotaUnit, setQuotaUnit] = useState<string>('GB')

  // AI 决策弹窗
  const [isLLMDialogOpen, setIsLLMDialogOpen] = useState(false)
  const [llmUser, setLlmUser] = useState<string>('')
  const [llmJobId, setLlmJobId] = useState<string | null>(null)
  const [llmDecisionJobId, setLlmDecisionJobId] = useState<string | null>(null)
  const [llmResult, setLlmResult] = useState<LLMDecisionResponse | null>(null)

  // 获取所有用户空间数据
  const userSpacesQuery = useQuery({
    queryKey: ['admin', 'user-spaces'],
    queryFn: () =>
      apiAdminGetUserSpaces(1, 1000).then((res: IResponse<PagedUserSpaces>) => res.data.items),
    staleTime: 5 * 60 * 1000,
  })

  // 设置配额 mutation
  const setQuotaMutation = useMutation({
    mutationFn: ({ user, quota }: { user: string; quota: number }) =>
      apiAdminSetUserSpaceQuota(user, quota),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'user-spaces'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'storage-decisions'] })
      toast('理论配额设置成功')
      setIsQuotaDialogOpen(false)
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '设置配额失败'))
    },
  })

  // 启动 AI 决策任务
  const llmStartMutation = useMutation({
    mutationFn: (user: string) => apiAdminStartLLMDecision(user),
    onSuccess: (res) => {
      setLlmJobId(res.data.job_id)
      setLlmDecisionJobId(res.data.job_id)
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, 'AI 分析启动失败'))
      setIsLLMDialogOpen(false)
    },
  })

  // 应用临时扩容
  const applyExpansionMutation = useMutation({
    mutationFn: ({
      user,
      expandBytes,
      freezeNewJobs,
      decisionJobId,
    }: {
      user: string
      expandBytes: number
      freezeNewJobs: boolean
      decisionJobId?: string
    }) => apiAdminApplyExpansion(user, expandBytes, freezeNewJobs, decisionJobId),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'user-spaces'] })
      toast(
        `已为 ${res.data.user} 临时扩容至 ${res.data.new_quota_formatted}（原配额 ${res.data.original_quota_formatted}）`
      )
      setIsLLMDialogOpen(false)
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '应用扩容失败'))
    },
  })

  // 还原配额
  const revertExpansionMutation = useMutation({
    mutationFn: (user: string) => apiAdminRevertExpansion(user),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'user-spaces'] })
      if (res.data.jobs_unfrozen) {
        toast(`已还原 ${res.data.user} 配额至 ${res.data.reverted_quota_formatted}，作业限制已解除`)
      } else {
        toast.warning(
          `已还原 ${res.data.user} 配额至 ${res.data.reverted_quota_formatted}，但存储用量仍超过理论配额，作业仍处于冻结状态`
        )
      }
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '还原配额失败'))
    },
  })

  // 手动解冻作业
  const unfreezeJobsMutation = useMutation({
    mutationFn: (user: string) => apiAdminUnfreezeJobs(user),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'user-spaces'] })
      toast(`已解除 ${res.data.user} 的作业创建限制`)
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '解冻失败'))
    },
  })

  // 轮询任务状态（每 3 秒，直到完成）
  const freezeJobsMutation = useMutation({
    mutationFn: ({ user, decisionJobId }: { user: string; decisionJobId?: string }) =>
      apiAdminFreezeJobs(user, decisionJobId),
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'user-spaces'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'storage-decisions'] })
      toast(`已冻结 ${res.data.user} 的新作业创建权限`)
      setIsLLMDialogOpen(false)
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '冻结作业失败'))
    },
  })

  const llmStatusQuery = useQuery({
    queryKey: ['llm-decision', llmUser, llmJobId],
    queryFn: () => apiAdminGetLLMDecisionStatus(llmUser, llmJobId!),
    enabled: !!llmJobId,
    refetchInterval: (query) => {
      const status = query.state.data?.data?.status
      return status === 'done' || status === 'error' ? false : 3000
    },
  })

  // 轮询到结果后更新 UI
  const jobStatus = llmStatusQuery.data?.data
  useEffect(() => {
    if (jobStatus?.status === 'done' && jobStatus.result) {
      setLlmResult(jobStatus.result)
      setLlmJobId(null)
      void queryClient.invalidateQueries({ queryKey: ['admin', 'storage-decisions'] })
    }
  }, [jobStatus, queryClient])

  const convertToBytes = (value: number, unit: string): number => {
    switch (unit) {
      case 'B':
        return Math.round(value)
      case 'KB':
        return Math.round(value * 1024)
      case 'MB':
        return Math.round(value * 1024 * 1024)
      case 'GB':
        return Math.round(value * 1024 * 1024 * 1024)
      case 'TB':
        return Math.round(value * 1024 * 1024 * 1024 * 1024)
      default:
        return Math.round(value)
    }
  }

  const alignQuotaInput = () => {
    if (quotaUnit === 'unlimited') return
    const normalized = normalizeQuotaDisplay(quotaValue, quotaUnit)
    setQuotaValue(normalized.value)
    setQuotaUnit(normalized.unit)
  }

  const handleSetQuota = () => {
    if (!selectedUser) return
    alignQuotaInput()
    const normalized = normalizeQuotaDisplay(quotaValue, quotaUnit)
    const quotaInBytes =
      normalized.unit === 'unlimited' ? -1 : convertToBytes(normalized.value, normalized.unit)
    setQuotaMutation.mutate({ user: selectedUser.user, quota: quotaInBytes })
  }

  const openSetQuotaDialog = (user: UserSpace) => {
    setSelectedUser(user)
    if (user.quota === -1) {
      setQuotaValue(0)
      setQuotaUnit('unlimited')
    } else if (user.quota >= 1024 ** 4) {
      setQuotaValue(Math.round(user.quota / 1024 ** 4))
      setQuotaUnit('TB')
    } else if (user.quota >= 1024 ** 3) {
      setQuotaValue(Math.round(user.quota / 1024 ** 3))
      setQuotaUnit('GB')
    } else if (user.quota >= 1024 ** 2) {
      setQuotaValue(Math.round(user.quota / 1024 ** 2))
      setQuotaUnit('MB')
    } else if (user.quota >= 1024) {
      setQuotaValue(Math.round(user.quota / 1024))
      setQuotaUnit('KB')
    } else {
      setQuotaValue(user.quota)
      setQuotaUnit('B')
    }
    setIsQuotaDialogOpen(true)
  }

  const openLLMDialog = (user: UserSpace) => {
    setLlmUser(user.user)
    setLlmResult(null)
    setLlmJobId(null)
    setLlmDecisionJobId(null)
    setIsLLMDialogOpen(true)
    llmStartMutation.mutate(user.user)
  }

  const columns: ColumnDef<UserSpace>[] = [
    {
      accessorKey: 'user',
      header: ({ column }) => <DataTableColumnHeader column={column} title="用户" />,
      cell: ({ row }) => (
        <span className="flex items-center gap-2">
          {row.getValue('user')}
          {row.original.jobs_frozen && (
            <Badge className="border-red-300 bg-red-100 text-xs text-red-700">作业已冻结</Badge>
          )}
          {row.original.shrink_stage === 'buffer_reduction' && (
            <Badge className="border-sky-300 bg-sky-100 text-xs text-sky-700">缩容缓冲期</Badge>
          )}
        </span>
      ),
    },
    {
      accessorKey: 'size',
      header: ({ column }) => <DataTableColumnHeader column={column} title="已用空间" />,
      cell: ({ row }) => row.original.formatted,
    },
    {
      accessorKey: 'shrink_stage',
      header: ({ column }) => <DataTableColumnHeader column={column} title="当前缩容状态" />,
      cell: ({ row }) => {
        const stage = row.original.shrink_stage
        if (!stage) return '无'
        const badgeClass =
          stage === 'buffer_reduction'
            ? 'border-sky-300 bg-sky-100 text-xs text-sky-700'
            : 'border-amber-300 bg-amber-100 text-xs text-amber-700'
        return <Badge className={badgeClass}>{shrinkStageLabel(stage)}</Badge>
      },
    },
    {
      accessorKey: 'original_quota',
      header: ({ column }) => <DataTableColumnHeader column={column} title="理论配额" />,
      cell: ({ row }) => {
        const { quota, quota_formatted, is_expanded, original_quota_formatted } = row.original
        if (quota === -1 && !is_expanded) return '无限制'
        return is_expanded ? original_quota_formatted : quota_formatted
      },
    },
    {
      accessorKey: 'quota',
      header: ({ column }) => <DataTableColumnHeader column={column} title="现配额" />,
      cell: ({ row }) => {
        const { quota, quota_formatted, is_expanded } = row.original
        if (quota === -1) return '无限制'
        return is_expanded ? (
          <span className="flex items-center gap-1">
            {quota_formatted}
            <Badge className="border-orange-300 bg-orange-100 text-xs text-orange-700">
              临时扩容
            </Badge>
          </span>
        ) : (
          quota_formatted
        )
      },
    },
    {
      accessorKey: 'usage_ratio',
      header: ({ column }) => <DataTableColumnHeader column={column} title="使用率" />,
      cell: ({ row }) => {
        const { size, quota } = row.original
        if (quota === -1 || quota === 0) return '—'
        const ratio = (size / quota) * 100
        const color =
          ratio >= 90
            ? 'text-red-500 font-semibold'
            : ratio >= 70
              ? 'text-yellow-500'
              : 'text-green-600'
        return <span className={color}>{ratio.toFixed(1)}%</span>
      },
    },
    {
      accessorKey: 'actions',
      header: '操作',
      cell: ({ row }) => {
        const user = row.original
        return (
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={() => openSetQuotaDialog(user)}>
              设置配额
            </Button>
            {user.is_expanded && (
              <Button
                variant="ghost"
                size="sm"
                className="text-orange-600 hover:text-orange-700"
                disabled={revertExpansionMutation.isPending}
                onClick={() => revertExpansionMutation.mutate(user.user)}
              >
                还原配额
              </Button>
            )}
            {user.jobs_frozen && (
              <Button
                variant="ghost"
                size="sm"
                className="text-red-600 hover:text-red-700"
                disabled={unfreezeJobsMutation.isPending}
                onClick={() => unfreezeJobsMutation.mutate(user.user)}
              >
                解冻作业
              </Button>
            )}
            <Button variant="ghost" size="sm" onClick={() => openLLMDialog(user)}>
              <BotMessageSquare className="mr-1 h-4 w-4" />
              AI 建议
            </Button>
          </div>
        )
      },
    },
  ]

  return (
    <>
      <DataTable
        info={{
          title: t('navigation.storageManagement'),
          description: '用户空间使用情况',
        }}
        storageKey="admin-storage"
        query={userSpacesQuery}
        columns={columns}
      />
      <StorageIndexPanel />
      <StorageDirectoryComparePanel />
      <StorageGovernancePanel />

      {/* 设置配额弹窗 */}
      <Dialog open={isQuotaDialogOpen} onOpenChange={setIsQuotaDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>设置理论配额 - {selectedUser?.user}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            {selectedUser?.is_expanded && (
              <p className="text-muted-foreground text-sm">
                该用户当前有临时扩容，此处修改的是理论配额，现配额 ({selectedUser.quota_formatted})
                保持不变，还原扩容时将恢复为新设的理论配额。
              </p>
            )}
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="quota" className="text-right">
                理论配额
              </Label>
              <Input
                id="quota"
                type="number"
                min={0}
                step="any"
                value={quotaValue}
                onChange={(e) => setQuotaValue(Number(e.target.value))}
                onBlur={alignQuotaInput}
                disabled={quotaUnit === 'unlimited'}
              />
              <Select value={quotaUnit} onValueChange={setQuotaUnit}>
                <SelectTrigger>
                  <SelectValue placeholder="选择单位" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unlimited">无限制</SelectItem>
                  <SelectItem value="TB">TB</SelectItem>
                  <SelectItem value="GB">GB</SelectItem>
                  <SelectItem value="MB">MB</SelectItem>
                  <SelectItem value="KB">KB</SelectItem>
                  <SelectItem value="B">B</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="text-muted-foreground text-sm">
              当前已用空间: {selectedUser?.formatted}
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              onClick={handleSetQuota}
              disabled={setQuotaMutation.isPending || !selectedUser}
            >
              {setQuotaMutation.isPending ? '设置中...' : '设置配额'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* AI 决策弹窗 */}
      <Dialog open={isLLMDialogOpen} onOpenChange={setIsLLMDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <BotMessageSquare className="h-5 w-5" />
              AI 存储建议 — {llmUser}
            </DialogTitle>
          </DialogHeader>

          {(llmStartMutation.isPending || !!llmJobId) && !llmResult && (
            <div className="text-muted-foreground flex flex-col items-center gap-3 py-8">
              <Loader2 className="h-8 w-8 animate-spin" />
              <p className="text-sm">AI 正在分析存储情况，请稍候（约 10~30 秒）...</p>
            </div>
          )}

          {llmResult && (
            <div className="space-y-4 py-2">
              <div className="flex items-center gap-3">
                <span className="w-24 text-sm font-medium">扩容建议</span>
                {llmResult.allow_expand ? (
                  <Badge className="border-green-300 bg-green-100 text-green-700">建议扩容</Badge>
                ) : (
                  <Badge className="border-red-300 bg-red-100 text-red-700">无需扩容</Badge>
                )}
              </div>

              {llmResult.allow_expand && llmResult.expand_bytes > 0 && (
                <div className="flex items-center gap-3">
                  <span className="w-24 text-sm font-medium">建议扩容量</span>
                  <span className="text-sm">{formatBytes(llmResult.expand_bytes)}</span>
                </div>
              )}

              <div className="flex items-center gap-3">
                <span className="w-24 text-sm font-medium">暂停新任务</span>
                {llmResult.freeze_new_jobs ? (
                  <Badge className="border-yellow-300 bg-yellow-100 text-yellow-700">
                    建议暂停
                  </Badge>
                ) : (
                  <Badge variant="outline">无需暂停</Badge>
                )}
              </div>

              <div className="space-y-1">
                <span className="text-sm font-medium">决策理由</span>
                <p className="text-muted-foreground bg-muted rounded-md p-3 text-sm leading-relaxed">
                  {llmResult.reason}
                </p>
              </div>
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsLLMDialogOpen(false)}>
              关闭
            </Button>
            {llmResult && (
              <Button
                variant="outline"
                onClick={() => {
                  setLlmResult(null)
                  setLlmJobId(null)
                  setLlmDecisionJobId(null)
                  llmStartMutation.mutate(llmUser)
                }}
                disabled={llmStartMutation.isPending || !!llmJobId}
              >
                重新分析
              </Button>
            )}
            {llmResult?.allow_expand && llmResult.expand_bytes > 0 && (
              <Button
                onClick={() =>
                  applyExpansionMutation.mutate({
                    user: llmUser,
                    expandBytes: llmResult.expand_bytes,
                    freezeNewJobs: llmResult.freeze_new_jobs,
                    decisionJobId: llmDecisionJobId ?? undefined,
                  })
                }
                disabled={applyExpansionMutation.isPending}
              >
                {applyExpansionMutation.isPending
                  ? '应用中...'
                  : `应用扩容 +${formatBytes(llmResult.expand_bytes)}${llmResult.freeze_new_jobs ? '（含冻结作业）' : ''}`}
              </Button>
            )}
            {llmResult?.freeze_new_jobs && !llmResult.allow_expand && (
              <Button
                onClick={() =>
                  freezeJobsMutation.mutate({
                    user: llmUser,
                    decisionJobId: llmDecisionJobId ?? undefined,
                  })
                }
                disabled={freezeJobsMutation.isPending}
              >
                {freezeJobsMutation.isPending ? '冻结中...' : '执行冻结'}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
