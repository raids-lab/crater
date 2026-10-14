import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { CursorList } from '@/components/query-table/cursor-list'

interface SnapshotListProps<T> {
  items: T[]
  renderItem: (item: T, index: number) => ReactNode
  getItemKey?: (item: T, index: number) => string | number
  hasMore: boolean
  observedAt?: string
  isLoading?: boolean
  isRefreshing?: boolean
  emptyText?: string
  onLoadMore: () => void
  onRefresh: () => void
}

/**
 * SnapshotList adds the consistency timestamp to CursorList without
 * introducing page numbers or a database-style total.
 */
export function SnapshotList<T>({ observedAt, ...props }: SnapshotListProps<T>) {
  return (
    <div className="space-y-2">
      <SnapshotStatus observedAt={observedAt} />
      <CursorList {...props} />
    </div>
  )
}

export function SnapshotStatus({ observedAt }: { observedAt?: string }) {
  const { t } = useTranslation()
  if (!observedAt) return null

  return (
    <p className="text-muted-foreground text-xs">
      {t('snapshotList.observedAt', { time: new Date(observedAt).toLocaleString() })}
    </p>
  )
}
