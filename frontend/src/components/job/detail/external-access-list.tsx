import { ExternalLink, GridIcon, Trash2 } from 'lucide-react'
import { ReactNode } from 'react'

import TipBadge from '@/components/badge/tip-badge'
import { CopyButton } from '@/components/button/copy-button'
import TooltipButton from '@/components/button/tooltip-button'
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

interface ExternalAccessListProps<T> {
  items: T[]
  renderItem: (item: T) => ReactNode
  isLoading?: boolean
  emptyIcon?: ReactNode
  emptyText?: string
  docsButton: ReactNode
  addButton: ReactNode
}

export function ExternalAccessList<T>({
  items,
  renderItem,
  isLoading = false,
  emptyIcon = <GridIcon className="h-6 w-6" />,
  emptyText = '暂无数据',
  docsButton,
  addButton,
}: ExternalAccessListProps<T>) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        {isLoading ? (
          <div className="text-muted-foreground py-6 text-center">加载中...</div>
        ) : !items || items.length === 0 ? (
          <div className="text-muted-foreground text-center">
            <div className="flex flex-col items-center justify-center pt-8">
              <div className="bg-muted mb-4 rounded-full p-3">{emptyIcon}</div>
              <p className="select-none">{emptyText}</p>
            </div>
          </div>
        ) : (
          items.map(renderItem)
        )}
      </div>
      <div className="flex justify-end gap-2">
        {docsButton}
        {addButton}
      </div>
    </div>
  )
}

export const ExternalAccessItem = ({
  name,
  port,
  url,
  fullURL,
  isDeleting,
  handleDeleteItem,
}: {
  name: string
  port: number
  url: string
  fullURL?: string
  isDeleting?: boolean
  handleDeleteItem: () => void
}) => (
  <div className="bg-sidebar/75 flex items-center space-x-2 rounded-md border p-3">
    <div className="ml-2 flex grow flex-col items-start justify-start gap-0.5">
      <div className="flex flex-row items-center justify-start gap-2">
        <p className="font-semibold">{name}</p>
        <TipBadge
          title={
            <span>
              端口:
              <span className="ml-0.5 font-mono">{port}</span>
            </span>
          }
        />
      </div>
      <div className="text-muted-foreground flex flex-row font-mono text-xs">{url}</div>
    </div>
    <TooltipButton
      variant="ghost"
      size="icon"
      className="hover:text-primary"
      onClick={() => {
        window.open(fullURL ?? url, '_blank')
      }}
      tooltipContent="访问链接"
    >
      <ExternalLink className="size-4" />
    </TooltipButton>
    <CopyButton content={url} />
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <TooltipButton
          variant="ghost"
          size="icon"
          className="hover:text-destructive"
          tooltipContent="删除"
          disabled={isDeleting}
        >
          <Trash2 className="size-4" />
        </TooltipButton>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>删除外部访问规则</AlertDialogTitle>
          <AlertDialogDescription>
            外部访问规则「{name}」<br />
            {port} → {url}
            <br />
            将被删除，请谨慎操作。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>取消</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={handleDeleteItem} disabled={isDeleting}>
            {isDeleting ? '删除中...' : '删除'}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
)
