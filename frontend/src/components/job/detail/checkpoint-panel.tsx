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
import { useNavigate } from '@tanstack/react-router'
import {
  ArchiveRestoreIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  ShieldCheckIcon,
  Trash2Icon,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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

import {
  JobCheckpoint,
  apiJobCheckpointCleanup,
  apiJobCheckpointDelete,
  apiJobCheckpointRestore,
  apiJobCheckpointScan,
  apiJobCheckpoints,
} from '@/services/api/vcjob'

import { formatBytes } from '@/utils/formatter'

import { cn } from '@/lib/utils'

interface CheckpointPanelProps {
  jobName: string
}

export default function CheckpointPanel({ jobName }: CheckpointPanelProps) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const queryKey = ['job', 'detail', jobName, 'checkpoints']

  const { data, isFetching } = useQuery({
    queryKey,
    queryFn: () => apiJobCheckpoints(jobName, true).then((res) => res.data),
  })

  const refreshQueries = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey }),
      queryClient.invalidateQueries({ queryKey: ['job', 'detail', jobName] }),
    ])
  }

  const scanMutation = useMutation({
    mutationFn: () => apiJobCheckpointScan(jobName),
    onSuccess: async () => {
      toast.success('Checkpoint 扫描完成')
      await refreshQueries()
    },
  })

  const cleanupMutation = useMutation({
    mutationFn: () =>
      apiJobCheckpointCleanup(jobName, {
        keepLast: data?.quota.maxToKeep || data?.checkpoint?.maxToKeep || 3,
      }),
    onSuccess: async (res) => {
      toast.success(`清理完成，释放 ${formatBytes(res.data.reclaimedBytes)}`)
      await refreshQueries()
    },
  })

  const restoreMutation = useMutation({
    mutationFn: (checkpoint: JobCheckpoint) =>
      apiJobCheckpointRestore(jobName, checkpoint.ID, {
        name: `${checkpoint.name}-resume`,
      }),
    onSuccess: async (res) => {
      toast.success(`已提交恢复作业 ${res.data.jobName}`)
      await queryClient.invalidateQueries({ queryKey: ['job'] })
      navigate({
        to: '/portal/jobs/detail/$name',
        params: { name: res.data.jobName },
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (checkpoint: JobCheckpoint) => apiJobCheckpointDelete(jobName, checkpoint.ID),
    onSuccess: async () => {
      toast.success('Checkpoint 已删除')
      await refreshQueries()
    },
  })

  const checkpoints = data?.items ?? []
  const maxToKeep = data?.quota.maxToKeep || data?.checkpoint?.maxToKeep || 0
  const lastScannedAt = data?.lastScannedAt ? new Date(data.lastScannedAt).toLocaleString() : '-'

  return (
    <div className="flex flex-col gap-4 p-4 md:p-6">
      <div className="grid gap-3 md:grid-cols-4">
        <Metric label="Latest" value={data?.latest?.name ?? '-'} mono />
        <Metric label="数量" value={`${data?.quota.currentCount ?? 0}/${maxToKeep || '-'}`} />
        <Metric
          label="容量"
          value={`${formatBytes(data?.quota.currentBytes ?? 0)}${
            data?.quota.maxBytes ? ` / ${formatBytes(data.quota.maxBytes)}` : ''
          }`}
        />
        <Metric label="上次扫描" value={lastScannedAt} />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="text-muted-foreground flex items-center gap-2 text-sm">
          <ShieldCheckIcon className="size-4" />
          <span>
            保留策略按最新 checkpoint 优先，超额数量 {data?.quota.excessCount ?? 0}
            {data?.quota.excessBytes ? `，超额容量 ${formatBytes(data.quota.excessBytes)}` : ''}
          </span>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            onClick={() => scanMutation.mutate()}
            disabled={scanMutation.isPending || isFetching}
          >
            <RefreshCwIcon className={cn('size-4', scanMutation.isPending && 'animate-spin')} />
            扫描
          </Button>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="outline"
                disabled={cleanupMutation.isPending || checkpoints.length === 0}
              >
                <Trash2Icon className="size-4" />
                清理
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>清理 checkpoint</AlertDialogTitle>
                <AlertDialogDescription>
                  将保留最新 {maxToKeep || 3} 个 checkpoint，删除超出配额的历史项。
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction onClick={() => cleanupMutation.mutate()}>
                  清理
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>名称</TableHead>
            <TableHead>Step</TableHead>
            <TableHead>大小</TableHead>
            <TableHead>更新时间</TableHead>
            <TableHead>路径</TableHead>
            <TableHead className="text-right">操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {checkpoints.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} className="text-muted-foreground h-24 text-center">
                暂无 checkpoint
              </TableCell>
            </TableRow>
          )}
          {checkpoints.map((checkpoint) => (
            <TableRow key={checkpoint.ID}>
              <TableCell>
                <div className="flex items-center gap-2">
                  <span className="font-medium">{checkpoint.name}</span>
                  {checkpoint.latest && <Badge variant="secondary">latest</Badge>}
                </div>
              </TableCell>
              <TableCell>{checkpoint.step >= 0 ? checkpoint.step : '-'}</TableCell>
              <TableCell>{formatBytes(checkpoint.sizeBytes)}</TableCell>
              <TableCell>{new Date(checkpoint.modTime).toLocaleString()}</TableCell>
              <TableCell className="max-w-[360px] truncate font-mono text-xs">
                {checkpoint.path}
              </TableCell>
              <TableCell>
                <div className="flex justify-end gap-2">
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="secondary" size="sm" disabled={restoreMutation.isPending}>
                        <ArchiveRestoreIcon className="size-4" />
                        恢复
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>从 checkpoint 恢复</AlertDialogTitle>
                        <AlertDialogDescription>
                          将基于当前作业配置提交一个新作业，并设置恢复路径为 {checkpoint.path}。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction onClick={() => restoreMutation.mutate(checkpoint)}>
                          提交恢复作业
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="outline" size="sm" disabled={deleteMutation.isPending}>
                        <Trash2Icon className="size-4" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>删除 checkpoint</AlertDialogTitle>
                        <AlertDialogDescription>
                          将删除 {checkpoint.name} 对应的存储目录，并记录审计日志。
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => deleteMutation.mutate(checkpoint)}
                        >
                          删除
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {isFetching && (
        <div className="text-muted-foreground flex items-center gap-2 text-sm">
          <RotateCcwIcon className="size-4 animate-spin" />
          正在同步 checkpoint 列表
        </div>
      )}
    </div>
  )
}

function Metric({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="border-border rounded-md border px-3 py-2">
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className={cn('mt-1 truncate text-sm font-medium', mono && 'font-mono')}>{value}</div>
    </div>
  )
}
