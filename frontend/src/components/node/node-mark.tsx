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
import { ColumnDef } from '@tanstack/react-table'
import { Plus, Trash2Icon } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogTrigger } from '@/components/ui/dialog'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { AddNodeMarkDialog } from '@/components/node/add-node-mark-dialog'
import { DeleteConfirmDialog } from '@/components/node/delete-confirm-dialog'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  type IClusterNodeAnnotation,
  type IClusterNodeLabel,
  type IClusterNodeTaint,
  TaintEffect,
  apiAddNodeAnnotation,
  apiAddNodeLabel,
  apiAddNodeTaint,
  apiDeleteNodeAnnotation,
  apiDeleteNodeLabel,
  apiDeleteNodeTaint,
  apiGetNodeMark,
} from '@/services/api/cluster'

import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

export enum DialogType {
  LABEL = 'label',
  ANNOTATION = 'annotation',
  TAINT = 'taint',
}

interface NodeMarkListProps {
  nodeName: string
}

// 标记类型配置
interface MarkTypeConfig {
  type: DialogType
  title: string
  addButtonText: string
  searchPlaceholder: string
  columns: {
    key: string
    value: string
    effect?: string
  }
  hasEffect: boolean
  maxLengths: {
    key: number
    value: number
    effect?: number
  }
}

const MARK_TYPE_CONFIGS: Record<DialogType, MarkTypeConfig> = {
  [DialogType.LABEL]: {
    type: DialogType.LABEL,
    title: 'Labels',
    addButtonText: '添加 Label 标签',
    searchPlaceholder: '搜索Label Key',
    columns: {
      key: 'Label名',
      value: 'Label值',
    },
    hasEffect: false,
    maxLengths: {
      key: 50,
      value: 50,
    },
  },
  [DialogType.ANNOTATION]: {
    type: DialogType.ANNOTATION,
    title: 'Annotations',
    addButtonText: '添加 Annotation 标注',
    searchPlaceholder: '搜索Annotation Key',
    columns: {
      key: 'Annotation名',
      value: 'Annotation值',
    },
    hasEffect: false,
    maxLengths: {
      key: 30,
      value: 50,
    },
  },
  [DialogType.TAINT]: {
    type: DialogType.TAINT,
    title: 'Taints',
    addButtonText: '添加 Taint 污点',
    searchPlaceholder: '搜索Taint Key',
    columns: {
      key: 'Taint名',
      value: 'Taint值',
      effect: 'Taint效果',
    },
    hasEffect: true,
    maxLengths: {
      key: 30,
      value: 50,
      effect: 20,
    },
  },
}

// 截断显示组件，带复制功能
function TruncatedCell({ value, maxLength = 30 }: { value: string; maxLength?: number }) {
  const { handleCopy } = useCopyToClipboard({ text: value })
  const shouldTruncate = value.length > maxLength
  const displayValue = shouldTruncate ? `${value.slice(0, maxLength)}...` : value

  if (shouldTruncate) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className="hover:text-primary cursor-pointer truncate font-mono text-sm"
              onClick={handleCopy}
            >
              {displayValue}
            </span>
          </TooltipTrigger>
          <TooltipContent side="top" className="max-w-md">
            <p className="text-xs break-all">{value}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  return (
    <span className="hover:text-primary cursor-pointer font-mono text-sm" onClick={handleCopy}>
      {value}
    </span>
  )
}

// 通用的节点标记管理组件
interface GenericNodeMarkProps {
  nodeName: string
  config: MarkTypeConfig
  standalone?: boolean
}

type NodeMarkItem = IClusterNodeLabel | IClusterNodeAnnotation | IClusterNodeTaint

function GenericNodeMark({ nodeName, config, standalone = false }: GenericNodeMarkProps) {
  const queryClient = useQueryClient()

  // 删除确认对话框状态
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean
    item: NodeMarkItem | null
  }>({
    open: false,
    item: null,
  })

  // 查询数据
  const query = useQuery({
    queryKey: ['nodes', nodeName, 'mark', config.type + 's'],
    queryFn: async () => {
      const res = await apiGetNodeMark(nodeName)
      const data = res.data
      switch (config.type) {
        case DialogType.LABEL:
          return data?.labels || []
        case DialogType.ANNOTATION:
          return data?.annotations || []
        case DialogType.TAINT:
          return data?.taints || []
        default:
          return []
      }
    },
    enabled: !!nodeName,
  })

  // 删除mutation
  const { mutate: deleteMutate } = useMutation({
    mutationFn: (item: NodeMarkItem) => {
      switch (config.type) {
        case DialogType.LABEL:
          return apiDeleteNodeLabel(nodeName, item as IClusterNodeLabel)
        case DialogType.ANNOTATION:
          return apiDeleteNodeAnnotation(nodeName, item as IClusterNodeAnnotation)
        case DialogType.TAINT:
          return apiDeleteNodeTaint(nodeName, item as IClusterNodeTaint)
        default:
          throw new Error('Unknown type')
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes', nodeName, 'mark'] })
      toast.success(`${config.columns.key.replace('名', '')}删除成功`)
    },
  })

  // 添加mutation
  const { mutate: addMutate } = useMutation({
    mutationFn: (item: NodeMarkItem) => {
      switch (config.type) {
        case DialogType.LABEL:
          return apiAddNodeLabel(nodeName, item as IClusterNodeLabel)
        case DialogType.ANNOTATION:
          return apiAddNodeAnnotation(nodeName, item as IClusterNodeAnnotation)
        case DialogType.TAINT:
          return apiAddNodeTaint(nodeName, item as IClusterNodeTaint)
        default:
          throw new Error('Unknown type')
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes', nodeName, 'mark'] })
      toast.success(`${config.columns.key.replace('名', '')}添加成功`)
    },
  })

  const handleDelete = useCallback(
    (item: NodeMarkItem) => {
      setDeleteDialog({
        open: true,
        item,
      })
    },
    [setDeleteDialog]
  )

  const handleAdd = useCallback(
    (key: string, value: string, effect?: string) => {
      const item = { key, value } as NodeMarkItem
      if (config.hasEffect && effect) {
        ;(item as IClusterNodeTaint).effect = effect
      }
      addMutate(item)
    },
    [addMutate, config.hasEffect]
  )

  const handleDeleteConfirm = useCallback(() => {
    if (deleteDialog.item) {
      deleteMutate(deleteDialog.item)
    }
  }, [deleteDialog.item, deleteMutate])

  // 列定义
  const columns: ColumnDef<NodeMarkItem>[] = useMemo(() => {
    const cols: ColumnDef<NodeMarkItem>[] = [
      {
        accessorKey: 'key',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={config.columns.key} />
        ),
        cell: ({ getValue }) => (
          <TruncatedCell value={getValue<string>()} maxLength={config.maxLengths.key} />
        ),
      },
      {
        accessorKey: 'value',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={config.columns.value} />
        ),
        cell: ({ getValue }) => (
          <TruncatedCell value={getValue<string>()} maxLength={config.maxLengths.value} />
        ),
      },
    ]

    if (config.hasEffect) {
      cols.push({
        accessorKey: 'effect',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={config.columns.effect!} />
        ),
        cell: ({ getValue }) => (
          <TruncatedCell value={getValue<string>()} maxLength={config.maxLengths.effect!} />
        ),
      })
    }

    cols.push({
      id: 'actions',
      header: '操作',
      cell: ({ row }) => (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => handleDelete(row.original)}
          className="h-8 w-8 p-0"
        >
          <Trash2Icon className="text-destructive h-4 w-4" />
        </Button>
      ),
      enableSorting: false,
    })

    return cols
  }, [config, handleDelete])

  // Toolbar配置
  const toolbarConfig: DataTableToolbarConfig = useMemo(() => {
    const filterOptions: {
      key: string
      title: string
      option: { value: string; label: string }[]
      defaultValues?: string[]
    }[] = []

    if (config.hasEffect) {
      filterOptions.push({
        key: 'effect',
        title: 'Effect',
        option: [
          { value: TaintEffect.NoSchedule, label: TaintEffect.NoSchedule },
          { value: TaintEffect.PreferNoSchedule, label: TaintEffect.PreferNoSchedule },
          { value: TaintEffect.NoExecute, label: TaintEffect.NoExecute },
        ],
        defaultValues: [],
      })
    }

    return {
      filterInput: {
        placeholder: config.searchPlaceholder,
        key: 'key',
      },
      filterOptions,
      getHeader: (key: string) => {
        switch (key) {
          case 'key':
            return config.columns.key
          case 'value':
            return config.columns.value
          case 'effect':
            return config.columns.effect || key
          default:
            return key
        }
      },
    }
  }, [config])

  const content = (
    <>
      <DataTable
        storageKey={`node-${config.type}s`}
        query={query}
        columns={columns}
        toolbarConfig={toolbarConfig}
      />

      <DeleteConfirmDialog
        open={deleteDialog.open}
        onOpenChange={(open) => setDeleteDialog((prev) => ({ ...prev, open }))}
        onConfirm={handleDeleteConfirm}
        title={`删除${config.columns.key.replace('名', '')}`}
        itemInfo={{
          key: deleteDialog.item?.key || '',
          value: deleteDialog.item?.value || '',
          effect: config.hasEffect ? (deleteDialog.item as IClusterNodeTaint)?.effect : undefined,
        }}
        type={config.type}
      />
    </>
  )

  if (standalone) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium">{config.title}</h3>
          <Dialog>
            <DialogTrigger asChild>
              <Button size="sm" className="flex items-center gap-2">
                <Plus className="size-4" />
                {config.addButtonText}
              </Button>
            </DialogTrigger>
            <DialogContent>
              <AddNodeMarkDialog type={config.type} onAdd={handleAdd} />
            </DialogContent>
          </Dialog>
        </div>
        {content}
      </div>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-lg font-medium">{config.title}</CardTitle>
        <Dialog>
          <DialogTrigger asChild>
            <Button size="sm" className="flex items-center gap-2">
              <Plus className="size-4" />
              添加
            </Button>
          </DialogTrigger>
          <DialogContent>
            <AddNodeMarkDialog type={config.type} onAdd={handleAdd} />
          </DialogContent>
        </Dialog>
      </CardHeader>
      <CardContent>{content}</CardContent>
    </Card>
  )
}

// 将三个表格同时展示在一个页面中，预留出来的接口
export function NodeMarkList({ nodeName }: NodeMarkListProps) {
  return (
    <div className="space-y-6">
      <GenericNodeMark nodeName={nodeName} config={MARK_TYPE_CONFIGS[DialogType.LABEL]} />
      <GenericNodeMark nodeName={nodeName} config={MARK_TYPE_CONFIGS[DialogType.ANNOTATION]} />
      <GenericNodeMark nodeName={nodeName} config={MARK_TYPE_CONFIGS[DialogType.TAINT]} />
    </div>
  )
}

// 单独的Labels组件
export function NodeLabels({ nodeName }: NodeMarkListProps) {
  return (
    <GenericNodeMark nodeName={nodeName} config={MARK_TYPE_CONFIGS[DialogType.LABEL]} standalone />
  )
}

// 单独的Annotations组件
export function NodeAnnotations({ nodeName }: NodeMarkListProps) {
  return (
    <GenericNodeMark
      nodeName={nodeName}
      config={MARK_TYPE_CONFIGS[DialogType.ANNOTATION]}
      standalone
    />
  )
}

// 单独的Taints组件
export function NodeTaints({ nodeName }: NodeMarkListProps) {
  return (
    <GenericNodeMark nodeName={nodeName} config={MARK_TYPE_CONFIGS[DialogType.TAINT]} standalone />
  )
}

export default NodeMarkList
