import { useMutation, useQuery } from '@tanstack/react-query'
import { ChevronRight, RefreshCcw, Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
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

import { FileSelectDialog } from '@/components/file/file-select-dialog'

import {
  MetadataFolderCompareFile,
  apiAdminCompareMetadataFolders,
  apiAdminGetMetadataFolderCompareJob,
} from '@/services/api/storage'

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
      return '未开始'
    case 'pending':
      return '待处理'
    case 'running':
      return '运行中'
    case 'done':
      return '已完成'
    case 'error':
      return '失败'
    default:
      return status || '未开始'
  }
}

export default function StorageDirectoryComparePanel() {
  const [isOpen, setIsOpen] = useState(false)
  const [compareType, setCompareType] = useState<'auto' | 'model' | 'dataset'>('auto')
  const [compareMode, setCompareMode] = useState<'optimized' | 'full_hash'>('optimized')
  const [compareLeftPath, setCompareLeftPath] = useState('')
  const [compareRightPath, setCompareRightPath] = useState('')
  const [lastCompareJobId, setLastCompareJobId] = useState('')

  const compareMutation = useMutation({
    mutationFn: () =>
      apiAdminCompareMetadataFolders({
        left_path: compareLeftPath,
        right_path: compareRightPath,
        compare_type: compareType,
        compare_mode: compareMode,
      }),
    onSuccess: (res) => {
      setLastCompareJobId(res.data.job_id)
      setIsOpen(true)
      toast.success(`已启动目录比对任务 ${res.data.job_id}`)
    },
    onError: (error: unknown) => {
      toast.error(getErrorMessage(error, '目录比对失败'))
    },
  })

  const compareJobQuery = useQuery({
    queryKey: ['admin', 'storage-index', 'compare-job', lastCompareJobId],
    queryFn: () => apiAdminGetMetadataFolderCompareJob(lastCompareJobId).then((res) => res.data),
    enabled: !!lastCompareJobId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'done' || status === 'error' ? false : 3000
    },
  })

  const compareResult = compareJobQuery.data?.result
  const mismatchedFiles = useMemo<MetadataFolderCompareFile[]>(
    () => compareResult?.files.filter((item) => !item.same) ?? [],
    [compareResult]
  )

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <Card>
        <CardHeader className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div className="space-y-1">
            <CardTitle className="flex flex-wrap items-center gap-2">
              <Search className="h-5 w-5" />
              <span>目录比对测试</span>
              {lastCompareJobId && (
                <Badge variant="outline" className={statusTone(compareJobQuery.data?.status)}>
                  {statusLabel(compareJobQuery.data?.status)}
                </Badge>
              )}
            </CardTitle>
            <CardDescription>
              独立选择两个目录，测试当前目录比对优化方案与完整 SHA256 基线的结果和耗时。
              默认收起，用到时再展开即可。
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={() => {
                void compareJobQuery.refetch()
              }}
              disabled={!lastCompareJobId || compareJobQuery.isFetching}
            >
              <RefreshCcw className="mr-2 h-4 w-4" />
              刷新结果
            </Button>
            <CollapsibleTrigger asChild>
              <Button variant="outline">
                <ChevronRight
                  className={`mr-2 h-4 w-4 transition-transform ${isOpen ? 'rotate-90' : ''}`}
                />
                {isOpen ? '收起' : '展开'}
              </Button>
            </CollapsibleTrigger>
          </div>
        </CardHeader>

        <CollapsibleContent>
          <CardContent className="space-y-6">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="max-w-xs space-y-2">
                <Label>比对类型</Label>
                <Select
                  value={compareType}
                  onValueChange={(value: 'auto' | 'model' | 'dataset') => setCompareType(value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="auto">自动</SelectItem>
                    <SelectItem value="model">模型</SelectItem>
                    <SelectItem value="dataset">数据集</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="max-w-xs space-y-2">
                <Label>比对方案</Label>
                <Select
                  value={compareMode}
                  onValueChange={(value: 'optimized' | 'full_hash') => setCompareMode(value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="optimized">优化方案</SelectItem>
                    <SelectItem value="full_hash">完整 SHA256 基线</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>左侧目录</Label>
                <FileSelectDialog
                  value={compareLeftPath}
                  allowSelectFile={false}
                  isadmin
                  title="选择左侧目录"
                  handleSubmit={(item) => setCompareLeftPath(`/${item.id.replace(/^\//, '')}`)}
                />
              </div>
              <div className="space-y-2">
                <Label>右侧目录</Label>
                <FileSelectDialog
                  value={compareRightPath}
                  allowSelectFile={false}
                  isadmin
                  title="选择右侧目录"
                  handleSubmit={(item) => setCompareRightPath(`/${item.id.replace(/^\//, '')}`)}
                />
              </div>
            </div>

            <div className="flex justify-end">
              <Button
                onClick={() => compareMutation.mutate()}
                disabled={
                  compareMutation.isPending ||
                  compareJobQuery.data?.status === 'running' ||
                  !compareLeftPath ||
                  !compareRightPath
                }
              >
                {compareMutation.isPending || compareJobQuery.data?.status === 'running'
                  ? '比对中...'
                  : '开始比对'}
              </Button>
            </div>

            {lastCompareJobId && (
              <div className="rounded-md border p-4">
                <div className="flex flex-wrap items-center gap-3">
                  <span className="text-sm font-medium">最近比对任务：</span>
                  <span className="font-mono text-xs">{lastCompareJobId}</span>
                  <Badge variant="outline" className={statusTone(compareJobQuery.data?.status)}>
                    {statusLabel(compareJobQuery.data?.status)}
                  </Badge>
                </div>
                {compareJobQuery.data?.error && (
                  <p className="mt-2 text-sm text-red-600">{compareJobQuery.data.error}</p>
                )}
              </div>
            )}

            {compareResult && (
              <div className="bg-muted/20 space-y-4 rounded-md border p-4">
                <div className="flex flex-wrap items-center gap-3">
                  <Badge
                    variant="outline"
                    className={compareResult.same ? statusTone('done') : statusTone('error')}
                  >
                    {compareResult.same ? '目录相同' : '目录不同'}
                  </Badge>
                  <Badge variant="outline">
                    {compareResult.compare_mode === 'full_hash' ? '完整 SHA256 基线' : '优化方案'}
                  </Badge>
                  <span className="text-muted-foreground text-sm">
                    总耗时 {compareResult.timing.total_ms} ms
                  </span>
                </div>

                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
                  <MetricCard
                    label="左侧关键文件"
                    value={String(compareResult.left_key_file_count)}
                  />
                  <MetricCard
                    label="右侧关键文件"
                    value={String(compareResult.right_key_file_count)}
                  />
                  <MetricCard label="精确匹配" value={String(compareResult.exact_match_count)} />
                  <MetricCard label="回退匹配" value={String(compareResult.fallback_match_count)} />
                  <MetricCard
                    label="通过校验"
                    value={`${compareResult.verified_file_count}/${compareResult.compared_file_count}`}
                  />
                  <MetricCard label="扫描耗时" value={`${compareResult.timing.scan_ms} ms`} />
                  <MetricCard label="配对耗时" value={`${compareResult.timing.pairing_ms} ms`} />
                  <MetricCard label="Header 耗时" value={`${compareResult.timing.header_ms} ms`} />
                  <MetricCard
                    label="采样哈希耗时"
                    value={`${compareResult.timing.sampled_hash_ms} ms`}
                  />
                  <MetricCard
                    label="完整 SHA256 耗时"
                    value={`${compareResult.timing.full_hash_ms} ms`}
                  />
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <div className="text-sm font-medium">仅左侧存在</div>
                    <div className="text-muted-foreground rounded-md border p-3 text-sm whitespace-pre-wrap">
                      {compareResult.missing_left.length === 0
                        ? '无'
                        : compareResult.missing_left.slice(0, 10).join('\n')}
                    </div>
                  </div>
                  <div className="space-y-2">
                    <div className="text-sm font-medium">仅右侧存在</div>
                    <div className="text-muted-foreground rounded-md border p-3 text-sm whitespace-pre-wrap">
                      {compareResult.missing_right.length === 0
                        ? '无'
                        : compareResult.missing_right.slice(0, 10).join('\n')}
                    </div>
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="text-sm font-medium">差异文件</div>
                  <div className="rounded-md border">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>文件</TableHead>
                          <TableHead>左侧路径</TableHead>
                          <TableHead>右侧路径</TableHead>
                          <TableHead>模式</TableHead>
                          <TableHead>原因</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {mismatchedFiles.length === 0 ? (
                          <TableRow>
                            <TableCell
                              colSpan={5}
                              className="text-muted-foreground h-16 text-center text-sm"
                            >
                              无差异文件
                            </TableCell>
                          </TableRow>
                        ) : (
                          mismatchedFiles.slice(0, 20).map((item, index) => (
                            <TableRow
                              key={`${item.left_relative_path}-${item.right_relative_path}-${index}`}
                            >
                              <TableCell>{item.file_name}</TableCell>
                              <TableCell className="max-w-[220px] truncate">
                                {item.left_relative_path}
                              </TableCell>
                              <TableCell className="max-w-[220px] truncate">
                                {item.right_relative_path}
                              </TableCell>
                              <TableCell>{item.verification_mode}</TableCell>
                              <TableCell>{item.reason || '-'}</TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
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
