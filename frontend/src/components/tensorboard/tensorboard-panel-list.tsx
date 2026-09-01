/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { ColumnDef } from '@tanstack/react-table'
import { LoaderCircleIcon, PlusIcon, TerminalIcon, Trash2Icon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import TensorboardStatusBadge, {
  getTensorboardStatusDescription,
} from '@/components/badge/tensorboard-status-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'

import {
  MAX_ACTIVE_TENSORBOARDS,
  TensorboardInfo,
  apiTensorboardDelete,
  apiTensorboardList,
} from '@/services/api/tensorboard'

import { showErrorToast } from '@/utils/toast'

function OpenTensorboardButton({ board }: { board: TensorboardInfo }) {
  const { t } = useTranslation()
  const button = (
    <Button
      variant="outline"
      size="sm"
      className="text-primary"
      disabled={board.status !== 'ready'}
      onClick={() => {
        const win = window.open(board.accessPath, '_blank', 'noopener,noreferrer')
        if (win) win.opener = null
      }}
    >
      <TerminalIcon className="mr-1 size-4" /> {t('tensorboard.list.open')}
    </Button>
  )

  if (board.status === 'ready') {
    return button
  }

  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex" tabIndex={0}>
            {button}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>{getTensorboardStatusDescription(board.status, board.statusReason, t)}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

export default function TensorboardPanelList() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [boardToDelete, setBoardToDelete] = useState<TensorboardInfo | null>(null)

  const tensorboardsQuery = useQuery({
    queryKey: ['tensorboards'],
    queryFn: apiTensorboardList,
    select: (res) => res.data,
    refetchInterval: 5000,
  })

  const {
    mutate: deleteBoard,
    isPending: isDeleting,
    variables: deletingBoardID,
  } = useMutation({
    mutationFn: apiTensorboardDelete,
    onSuccess: () => {
      setBoardToDelete(null)
      queryClient.invalidateQueries({ queryKey: ['tensorboards'] })
      toast.success(t('tensorboard.list.deleteSuccess'))
    },
    onError: (err: unknown) => {
      showErrorToast(err)
    },
  })

  const activeTensorboardCount = (tensorboardsQuery.data ?? []).filter(
    (board) => new Date(board.expiration).getTime() > Date.now()
  ).length
  const hasReachedLimit = activeTensorboardCount >= MAX_ACTIVE_TENSORBOARDS

  const columns = useMemo<ColumnDef<TensorboardInfo>[]>(
    () => [
      {
        accessorKey: 'id',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('tensorboard.list.id')} />
        ),
      },
      {
        accessorKey: 'createdAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('tensorboard.list.createdAt')} />
        ),
        cell: ({ row }) => <TimeDistance date={row.getValue('createdAt')} />,
      },
      {
        accessorKey: 'expiration',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('tensorboard.list.expiration')} />
        ),
        cell: ({ row }) => <TimeDistance date={row.getValue('expiration')} />,
      },
      {
        accessorKey: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('tensorboard.list.status')} />
        ),
        cell: ({ row }) => (
          <TensorboardStatusBadge
            status={row.original.status}
            statusReason={row.original.statusReason}
          />
        ),
      },
      {
        id: 'actions',
        header: t('tensorboard.list.actions'),
        cell: ({ row }) => {
          const tb = row.original
          return (
            <div className="flex flex-row space-x-2">
              <OpenTensorboardButton board={tb} />
              <Button
                variant="destructive"
                size="icon"
                aria-label={t('tensorboard.list.delete')}
                disabled={isDeleting}
                onClick={() => setBoardToDelete(tb)}
              >
                {isDeleting && deletingBoardID === tb.id ? (
                  <LoaderCircleIcon className="size-4 animate-spin" />
                ) : (
                  <Trash2Icon className="size-4" />
                )}
              </Button>
            </div>
          )
        },
      },
    ],
    [deletingBoardID, isDeleting, t]
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">{t('tensorboard.list.title')}</h2>
          <p className="text-muted-foreground">
            {t('tensorboard.list.description', {
              count: activeTensorboardCount,
              limit: MAX_ACTIVE_TENSORBOARDS,
            })}
          </p>
        </div>
        <div className="flex space-x-2">
          <Button
            disabled={hasReachedLimit}
            title={
              hasReachedLimit
                ? t('tensorboard.list.limitReached', { count: MAX_ACTIVE_TENSORBOARDS })
                : undefined
            }
            onClick={() => navigate({ to: '/portal/jobs/new/tensorboard' })}
          >
            <PlusIcon className="mr-2 size-4" /> {t('tensorboard.list.create')}
          </Button>
        </div>
      </div>

      <DataTable<TensorboardInfo, unknown>
        storageKey="tensorboards"
        query={tensorboardsQuery}
        columns={columns}
      />

      <AlertDialog
        open={boardToDelete !== null}
        onOpenChange={(open) => {
          if (!open && !isDeleting) {
            setBoardToDelete(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('tensorboard.list.deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('tensorboard.list.deleteDescription', { id: boardToDelete?.id ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={isDeleting || boardToDelete === null}
              onClick={(event) => {
                event.preventDefault()
                if (boardToDelete !== null && !isDeleting) {
                  deleteBoard(boardToDelete.id)
                }
              }}
            >
              {isDeleting ? t('tensorboard.list.deleting') : t('common.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
