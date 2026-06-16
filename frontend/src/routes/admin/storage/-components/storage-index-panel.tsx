import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronRight, RefreshCcw, Search } from 'lucide-react'
import { Fragment, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import SelectBox from '@/components/custom/select-box'

import { IAccount, apiAdminAccountList } from '@/services/api/account'
import { IUser, apiAdminUserList } from '@/services/api/admin/user'
import {
  MetadataCandidate,
  MetadataCandidateFile,
  apiAdminGetMetadataScan,
  apiAdminGetMetadataWorkspaceCandidateFiles,
  apiAdminGetMetadataWorkspaceCandidates,
  apiAdminGetMetadataWorkspaceOverview,
  apiAdminTriggerMetadataScan,
} from '@/services/api/storage'

type WorkspaceType = 'user' | 'account'

type WorkspaceOption = {
  value: string
  label: string
  labelNote?: string
}

function pathDepth(path: string): number {
  return path.split('/').filter(Boolean).length
}

function isSameOrDescendantPath(targetPath: string, rootPath: string): boolean {
  return targetPath === rootPath || targetPath.startsWith(`${rootPath}/`)
}

function aggregateCandidateRoots(items: MetadataCandidate[]): MetadataCandidate[] {
  const roots = [...items].sort((left, right) => {
    const depthDiff = pathDepth(left.targetPath) - pathDepth(right.targetPath)
    if (depthDiff !== 0) {
      return depthDiff
    }
    return left.targetPath.localeCompare(right.targetPath)
  })

  const collapsed: MetadataCandidate[] = []
  for (const item of roots) {
    if (collapsed.some((root) => isSameOrDescendantPath(item.targetPath, root.targetPath))) {
      continue
    }
    collapsed.push(item)
  }

  return collapsed.sort((left, right) => {
    if (left.candidateScore === right.candidateScore) {
      return left.targetPath.localeCompare(right.targetPath)
    }
    return right.candidateScore - left.candidateScore
  })
}

function formatBytes(bytes?: number): string {
  if (!Number.isFinite(bytes) || !bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** exponent).toFixed(1)} ${units[exponent]}`
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

function statusTone(status?: string): string {
  switch (status) {
    case 'done':
      return 'border-green-300 bg-green-100 text-green-700'
    case 'error':
      return 'border-red-300 bg-red-100 text-red-700'
    case 'running':
      return 'border-sky-300 bg-sky-100 text-sky-700'
    default:
      return 'border-amber-300 bg-amber-100 text-amber-700'
  }
}

function statusLabel(status?: string): string {
  switch (status) {
    case '':
    case undefined:
      return '未扫描'
    case 'pending':
      return '等待中'
    case 'running':
      return '运行中'
    case 'done':
      return '已完成'
    case 'error':
      return '失败'
    default:
      return status || '未扫描'
  }
}

function buildUserWorkspaceOption(user: IUser): WorkspaceOption {
  const secondary = [user.attributes.nickname, user.attributes.email].filter(Boolean).join(' | ')
  return {
    value: user.name,
    label: user.name,
    labelNote: secondary || undefined,
  }
}

function buildAccountWorkspaceOption(account: IAccount): WorkspaceOption {
  const secondary = [account.nickname, account.space].filter(Boolean).join(' | ')
  return {
    value: account.name,
    label: account.name,
    labelNote: secondary || undefined,
  }
}

export function CandidateFilesPanel({
  workspaceType,
  workspaceName,
  candidatePath,
}: {
  workspaceType: WorkspaceType
  workspaceName: string
  candidatePath: string
}) {
  const query = useQuery({
    queryKey: [
      'admin',
      'storage-index',
      'candidate-files',
      workspaceType,
      workspaceName,
      candidatePath,
    ],
    queryFn: () =>
      apiAdminGetMetadataWorkspaceCandidateFiles(workspaceType, workspaceName, candidatePath).then(
        (res) => res.data
      ),
    enabled: !!candidatePath,
  })

  const items = useMemo<MetadataCandidateFile[]>(() => query.data?.items ?? [], [query.data])

  if (query.isLoading) {
    return <div className="text-muted-foreground p-4 text-sm">正在加载关键文件...</div>
  }

  if (items.length === 0) {
    return <div className="text-muted-foreground p-4 text-sm">暂无关键文件结果。</div>
  }

  return (
    <div className="bg-background rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>文件</TableHead>
            <TableHead>相对路径</TableHead>
            <TableHead>公共文件</TableHead>
            <TableHead>大小</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((file) => (
            <TableRow key={file.id}>
              <TableCell>{file.fileName}</TableCell>
              <TableCell className="max-w-[260px] truncate">{file.relativePath}</TableCell>
              <TableCell className="max-w-[260px] truncate">
                {file.matchedPublicFile || '-'}
              </TableCell>
              <TableCell>{formatBytes(file.sizeBytes)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

export default function StorageIndexPanel() {
  const queryClient = useQueryClient()
  const [workspaceType, setWorkspaceType] = useState<WorkspaceType>('user')
  const [workspaceName, setWorkspaceName] = useState('')
  const [expandedCandidatePaths, setExpandedCandidatePaths] = useState<string[]>([])
  const [lastTriggered, setLastTriggered] = useState<{
    scanId: string
    workspaceType: WorkspaceType
    workspaceName: string
  } | null>(null)

  const effectiveWorkspaceName = workspaceName.trim()

  const handleWorkspaceTypeChange = (value: WorkspaceType) => {
    setWorkspaceType(value)
    setWorkspaceName('')
    setExpandedCandidatePaths([])
    setLastTriggered(null)
  }

  const handleWorkspaceNameChange = (value: string) => {
    setWorkspaceName(value)
    setExpandedCandidatePaths([])
    setLastTriggered(null)
  }

  const triggerMutation = useMutation({
    mutationFn: () =>
      apiAdminTriggerMetadataScan({
        workspace_type: workspaceType,
        workspace_name: effectiveWorkspaceName,
      }),
    onSuccess: (res) => {
      setLastTriggered({
        scanId: res.data.scan_id,
        workspaceType,
        workspaceName: res.data.workspace_name,
      })
      toast.success(`已启动扫描任务：${res.data.scan_id}`)
      void queryClient.invalidateQueries({ queryKey: ['admin', 'storage-index', 'overview'] })
      void queryClient.invalidateQueries({ queryKey: ['admin', 'storage-index', 'candidates'] })
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '触发元数据扫描失败。'))
    },
  })

  const scanQuery = useQuery({
    queryKey: ['admin', 'storage-index', 'scan', lastTriggered?.scanId],
    queryFn: () => apiAdminGetMetadataScan(lastTriggered!.scanId).then((res) => res.data),
    enabled: !!lastTriggered?.scanId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'done' || status === 'error' ? false : 3000
    },
  })

  const workspaceOptionsQuery = useQuery({
    queryKey: ['admin', 'storage-index', 'workspace-options', workspaceType],
    queryFn: async () => {
      if (workspaceType === 'user') {
        const res = await apiAdminUserList()
        return [...res.data]
          .sort((left, right) => left.name.localeCompare(right.name))
          .map(buildUserWorkspaceOption)
      }

      const res = await apiAdminAccountList()
      return [...res.data]
        .sort((left, right) => left.name.localeCompare(right.name))
        .map(buildAccountWorkspaceOption)
    },
    staleTime: 60_000,
  })

  const overviewQuery = useQuery({
    queryKey: ['admin', 'storage-index', 'overview', workspaceType, effectiveWorkspaceName],
    queryFn: () =>
      apiAdminGetMetadataWorkspaceOverview(workspaceType, effectiveWorkspaceName).then(
        (res) => res.data
      ),
    enabled: !!effectiveWorkspaceName,
  })

  const candidateQuery = useQuery({
    queryKey: ['admin', 'storage-index', 'candidates', workspaceType, effectiveWorkspaceName],
    queryFn: () =>
      apiAdminGetMetadataWorkspaceCandidates(workspaceType, effectiveWorkspaceName, 1, 20).then(
        (res) => res.data
      ),
    enabled: !!effectiveWorkspaceName,
  })

  const workspaceOptions = workspaceOptionsQuery.data ?? []
  const candidateItems = useMemo<MetadataCandidate[]>(
    () =>
      aggregateCandidateRoots(
        (candidateQuery.data?.items ?? []).filter((item) => item.status === 'verified')
      ),
    [candidateQuery.data]
  )
  const redundancyQuery = candidateQuery

  const toggleCandidate = (candidatePath: string) => {
    setExpandedCandidatePaths((current) =>
      current.includes(candidatePath)
        ? current.filter((item) => item !== candidatePath)
        : [...current, candidatePath]
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <CardTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            元数据索引扫描
          </CardTitle>
          <CardDescription>
            手动触发工作空间扫描，查看最新扫描状态，并检查相对公共基线的冗余候选结果。
          </CardDescription>
        </div>
        <Button
          variant="outline"
          onClick={() => {
            void overviewQuery.refetch()
            void candidateQuery.refetch()
            void scanQuery.refetch()
            void workspaceOptionsQuery.refetch()
          }}
          disabled={
            overviewQuery.isFetching ||
            candidateQuery.isFetching ||
            workspaceOptionsQuery.isFetching
          }
        >
          <RefreshCcw className="mr-2 h-4 w-4" />
          刷新结果
        </Button>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-4 md:grid-cols-[180px_1fr_auto]">
          <div className="space-y-2">
            <Label>工作空间类型</Label>
            <Select value={workspaceType} onValueChange={handleWorkspaceTypeChange}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">user</SelectItem>
                <SelectItem value="account">account</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>工作空间名称</Label>
            <SelectBox
              options={workspaceOptions}
              value={workspaceName ? [workspaceName] : []}
              onChange={(values) => handleWorkspaceNameChange(values.at(-1) ?? '')}
              placeholder={workspaceType === 'user' ? '请选择用户工作空间' : '请选择账户工作空间'}
              inputPlaceholder={workspaceType === 'user' ? '搜索用户工作空间' : '搜索账户工作空间'}
              emptyPlaceholder={
                workspaceType === 'user' ? '未找到用户工作空间' : '未找到账户工作空间'
              }
              className={
                workspaceOptionsQuery.isLoading || workspaceOptions.length === 0
                  ? 'pointer-events-none opacity-60'
                  : ''
              }
            />
            <p className="text-muted-foreground text-xs">
              {workspaceOptionsQuery.isLoading
                ? '正在加载工作空间列表...'
                : workspaceOptionsQuery.isError
                  ? '加载工作空间列表失败。'
                  : workspaceOptions.length > 0
                    ? `共 ${workspaceOptions.length} 个可选项`
                    : workspaceType === 'user'
                      ? '暂无可选用户工作空间'
                      : '暂无可选账户工作空间'}
            </p>
          </div>

          <div className="flex items-end">
            <Button
              onClick={() => triggerMutation.mutate()}
              disabled={
                triggerMutation.isPending ||
                !effectiveWorkspaceName ||
                workspaceOptionsQuery.isLoading
              }
            >
              {triggerMutation.isPending ? '启动中...' : '启动扫描'}
            </Button>
          </div>
        </div>

        {lastTriggered && (
          <div className="rounded-md border p-4">
            <div className="flex flex-wrap items-center gap-3">
              <span className="text-sm font-medium">最近触发任务：</span>
              <span className="font-mono text-xs">{lastTriggered.scanId}</span>
              <Badge variant="outline" className={statusTone(scanQuery.data?.status)}>
                {statusLabel(scanQuery.data?.status)}
              </Badge>
            </div>
            {scanQuery.data?.errorMessage && (
              <p className="mt-2 text-sm text-red-600">{scanQuery.data.errorMessage}</p>
            )}
          </div>
        )}

        {overviewQuery.data && (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <MetricCard label="最近扫描 ID" value={overviewQuery.data.last_scan_id || '-'} />
            <MetricCard
              label="最近扫描状态"
              value={statusLabel(overviewQuery.data.last_scan_status)}
            />
            <MetricCard label="目录数" value={String(overviewQuery.data.directory_count)} />
            <MetricCard label="候选目录数" value={String(overviewQuery.data.file_count)} />
            <MetricCard label="冗余命中数" value={String(overviewQuery.data.redundancy_count)} />
            <MetricCard
              label="冗余空间估算"
              value={formatBytes(overviewQuery.data.redundancy_bytes)}
            />
          </div>
        )}

        <div className="space-y-3">
          <h3 className="text-sm font-semibold">已验证冗余候选（前 20 项）</h3>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>类型</TableHead>
                  <TableHead>目标路径</TableHead>
                  <TableHead>公共路径</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>评分</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {candidateItems.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={5}
                      className="text-muted-foreground h-20 text-center text-sm"
                    >
                      {redundancyQuery.isLoading ? '正在加载冗余候选...' : '暂无冗余候选结果。'}
                    </TableCell>
                  </TableRow>
                ) : (
                  candidateItems.map((item) => {
                    const isExpanded = expandedCandidatePaths.includes(item.targetPath)
                    return (
                      <Fragment key={item.id}>
                        <TableRow
                          className="hover:bg-muted/40 cursor-pointer transition-colors"
                          onClick={() => toggleCandidate(item.targetPath)}
                        >
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <ChevronRight
                                className={`text-muted-foreground h-4 w-4 transition-transform ${
                                  isExpanded ? 'rotate-90' : ''
                                }`}
                              />
                              <span>{item.candidateType}</span>
                            </div>
                          </TableCell>
                          <TableCell className="max-w-[320px] truncate">
                            {item.targetPath}
                          </TableCell>
                          <TableCell className="max-w-[320px] truncate">
                            {item.publicPath}
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant="outline"
                              className={statusTone(
                                item.status === 'verified' ? 'done' : 'running'
                              )}
                            >
                              {item.status}
                            </Badge>
                          </TableCell>
                          <TableCell>{item.candidateScore.toFixed(1)}</TableCell>
                        </TableRow>
                        {isExpanded && (
                          <TableRow className="bg-muted/20">
                            <TableCell colSpan={5} className="p-0">
                              <div className="grid gap-4 p-4 md:grid-cols-2">
                                <div className="space-y-2">
                                  <div className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
                                    目标目录
                                  </div>
                                  <div className="bg-background rounded-md border p-3 font-mono text-xs">
                                    {item.targetPath}
                                  </div>
                                </div>
                                <div className="space-y-2">
                                  <div className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
                                    公共目录
                                  </div>
                                  <div className="bg-background rounded-md border p-3 font-mono text-xs">
                                    {item.publicPath || '-'}
                                  </div>
                                </div>
                                <div className="space-y-2 md:col-span-2">
                                  <div className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
                                    证据
                                  </div>
                                  <div className="bg-background text-muted-foreground rounded-md border p-3 text-sm">
                                    {item.evidence || '-'}
                                  </div>
                                </div>
                                <div className="space-y-2">
                                  <div className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
                                    候选评分
                                  </div>
                                  <div className="bg-background rounded-md border p-3 text-sm">
                                    {item.candidateScore.toFixed(1)}
                                  </div>
                                </div>
                                <div className="space-y-2">
                                  <div className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
                                    扫描批次
                                  </div>
                                  <div className="bg-background rounded-md border p-3 font-mono text-xs">
                                    {item.scanId}
                                  </div>
                                </div>
                                <div className="space-y-2 md:col-span-2">
                                  <CandidateFilesPanel
                                    workspaceType={workspaceType}
                                    workspaceName={effectiveWorkspaceName}
                                    candidatePath={item.targetPath}
                                  />
                                </div>
                              </div>
                            </TableCell>
                          </TableRow>
                        )}
                      </Fragment>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-muted/30 rounded-md border p-3">
      <div className="text-muted-foreground text-xs tracking-wide uppercase">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  )
}
