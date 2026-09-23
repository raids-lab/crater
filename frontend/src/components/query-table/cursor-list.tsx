import { RefreshCwIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

interface CursorListProps<T> {
  items: T[]
  renderItem: (item: T, index: number) => ReactNode
  getItemKey?: (item: T, index: number) => string | number
  hasMore: boolean
  isLoading?: boolean
  isRefreshing?: boolean
  emptyText?: string
  loadMoreLabel?: string
  refreshLabel?: string
  onLoadMore: () => void
  onRefresh: () => void
}

interface SnapshotControlsProps {
  hasMore: boolean
  isLoading?: boolean
  isRefreshing?: boolean
  loadMoreLabel?: string
  refreshLabel?: string
  onLoadMore: () => void
  onRefresh: () => void
}

export function SnapshotControls({
  hasMore,
  isLoading = false,
  isRefreshing = false,
  loadMoreLabel,
  refreshLabel,
  onLoadMore,
  onRefresh,
}: SnapshotControlsProps) {
  const { t } = useTranslation()

  return (
    <div className="flex items-center justify-between gap-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={onRefresh}
        disabled={isLoading || isRefreshing}
      >
        <RefreshCwIcon className={isRefreshing ? 'animate-spin' : undefined} />
        {refreshLabel ?? t('snapshotList.refresh')}
      </Button>
      {hasMore && (
        <Button type="button" size="sm" onClick={onLoadMore} disabled={isLoading}>
          {isLoading ? t('common.loading') : (loadMoreLabel ?? t('snapshotList.loadMore'))}
        </Button>
      )}
    </div>
  )
}

/**
 * CursorList is intentionally separate from RemoteDataTable: snapshot and
 * Kubernetes snapshots do not promise a stable database total or numbered pages.
 */
export function CursorList<T>({
  items,
  renderItem,
  getItemKey,
  hasMore,
  isLoading = false,
  isRefreshing = false,
  emptyText,
  loadMoreLabel,
  refreshLabel,
  onLoadMore,
  onRefresh,
}: CursorListProps<T>) {
  const { t } = useTranslation()

  return (
    <div className="space-y-3">
      {items.length === 0 && !isLoading ? (
        <div className="text-muted-foreground rounded-md border border-dashed p-8 text-center text-sm">
          {emptyText ?? t('common.noData')}
        </div>
      ) : (
        <div className="space-y-2">
          {items.map((item, index) => (
            <div key={getItemKey?.(item, index) ?? index}>{renderItem(item, index)}</div>
          ))}
        </div>
      )}
      <SnapshotControls
        hasMore={hasMore}
        isLoading={isLoading}
        isRefreshing={isRefreshing}
        loadMoreLabel={loadMoreLabel}
        refreshLabel={refreshLabel}
        onLoadMore={onLoadMore}
        onRefresh={onRefresh}
      />
    </div>
  )
}
