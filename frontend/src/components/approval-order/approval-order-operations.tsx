import { CheckIcon, EyeIcon, MoreHorizontalIcon, XIcon } from 'lucide-react'
import { ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import { type ApprovalOrder } from '@/services/api/approvalorder'

export interface ApprovalOrderActionConfig {
  view?: {
    show: boolean
    onClick: (order: ApprovalOrder) => void
    label?: string
  }
  approve?: {
    show: boolean
    onClick: (order: ApprovalOrder) => void
    label?: string
    disabled?: (order: ApprovalOrder) => boolean
  }
  reject?: {
    show: boolean
    onClick: (order: ApprovalOrder) => void
    label?: string
    disabled?: (order: ApprovalOrder) => boolean
  }
  custom?: {
    key: string
    label: string
    icon?: ReactNode
    onClick: (order: ApprovalOrder) => void
    disabled?: (order: ApprovalOrder) => boolean
    variant?: 'default' | 'destructive'
    separator?: boolean
  }[]
}

export interface ApprovalOrderOperationsProps {
  order: ApprovalOrder
  config: ApprovalOrderActionConfig
}

export function ApprovalOrderOperations({ order, config }: ApprovalOrderOperationsProps) {
  const hasActions =
    config.view?.show ||
    config.approve?.show ||
    config.reject?.show ||
    (config.custom && config.custom.length > 0)

  if (!hasActions) {
    return null
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" className="h-8 w-8 p-0">
          <span className="sr-only">打开菜单</span>
          <MoreHorizontalIcon className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {config.view?.show && (
          <DropdownMenuItem onClick={() => config.view?.onClick(order)}>
            <EyeIcon className="mr-2 h-4 w-4" />
            {config.view.label || '查看详情'}
          </DropdownMenuItem>
        )}

        {config.approve?.show && (
          <DropdownMenuItem
            onClick={() => config.approve?.onClick(order)}
            disabled={config.approve.disabled?.(order)}
          >
            <CheckIcon className="mr-2 h-4 w-4" />
            {config.approve.label || '批准'}
          </DropdownMenuItem>
        )}

        {config.reject?.show && (
          <DropdownMenuItem
            onClick={() => config.reject?.onClick(order)}
            disabled={config.reject.disabled?.(order)}
          >
            <XIcon className="mr-2 h-4 w-4" />
            {config.reject.label || '拒绝'}
          </DropdownMenuItem>
        )}

        {config.custom?.map((action, index) => (
          <div key={action.key}>
            {action.separator && index > 0 && <DropdownMenuSeparator />}
            <DropdownMenuItem
              onClick={() => action.onClick(order)}
              disabled={action.disabled?.(order)}
              className={action.variant === 'destructive' ? 'text-destructive' : ''}
            >
              {action.icon && <span className="mr-2">{action.icon}</span>}
              {action.label}
            </DropdownMenuItem>
          </div>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

// 常用的操作配置预设
export const createViewOnlyConfig = (
  onView: (order: ApprovalOrder) => void
): ApprovalOrderActionConfig => ({
  view: {
    show: true,
    onClick: onView,
  },
})

export const createAdminConfig = (
  onView: (order: ApprovalOrder) => void,
  onApprove: (order: ApprovalOrder) => void,
  onReject: (order: ApprovalOrder) => void,
  canManage: (order: ApprovalOrder) => boolean = () => true
): ApprovalOrderActionConfig => ({
  view: {
    show: true,
    onClick: onView,
  },
  approve: {
    show: true,
    onClick: onApprove,
    disabled: (order) => !canManage(order) || order.status !== 'Pending',
  },
  reject: {
    show: true,
    onClick: onReject,
    disabled: (order) => !canManage(order) || order.status !== 'Pending',
  },
})
