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
import { CheckCircle, Clock, Download, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'

type ModelDownloadStatus = 'Pending' | 'Downloading' | 'Ready' | 'Failed'

interface ModelDownloadStatusBadgeProps {
  status: ModelDownloadStatus
}

export function ModelDownloadStatusBadge({ status }: ModelDownloadStatusBadgeProps) {
  const config = {
    Pending: {
      icon: Clock,
      label: '等待中',
      variant: 'secondary' as const,
      className: '',
    },
    Downloading: {
      icon: Download,
      label: '下载中',
      variant: 'default' as const,
      className: '',
    },
    Ready: {
      icon: CheckCircle,
      label: '完成',
      variant: 'outline' as const,
      className: 'border-green-500 text-green-700 dark:text-green-400',
    },
    Failed: {
      icon: XCircle,
      label: '失败',
      variant: 'destructive' as const,
      className: '',
    },
  }

  const { icon: Icon, label, variant, className } = config[status] || config.Pending

  return (
    <Badge variant={variant} className={className}>
      <Icon className="h-3 w-3" />
      {label}
    </Badge>
  )
}
