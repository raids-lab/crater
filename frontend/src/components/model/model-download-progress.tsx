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
import { useTranslation } from 'react-i18next'

import { Progress } from '@/components/ui/progress'

import { ModelDownload } from '@/services/api/modeldownload'

import { formatFileSize } from '@/utils/file-size'

import { cn } from '@/lib/utils'

/**
 * 下载进度展示:
 * - 已知总大小(下载脚本上报 [TOTAL])时显示真实百分比进度条
 * - 总大小未知且下载中时显示不确定进度条(脉冲动画)
 * - 完成时显示 100% 与最终大小
 */
export function ModelDownloadProgress({
  download,
  className,
}: {
  download: Pick<ModelDownload, 'status' | 'sizeBytes' | 'downloadedBytes' | 'downloadSpeed'>
  className?: string
}) {
  const { t } = useTranslation()
  const { status, sizeBytes, downloadedBytes, downloadSpeed } = download

  if (status === 'Ready') {
    return (
      <div className={cn('w-full min-w-[140px] space-y-1', className)}>
        <Progress value={100} />
        <p className="text-muted-foreground text-xs">
          100% · {sizeBytes > 0 ? formatFileSize(sizeBytes) : '--'}
        </p>
      </div>
    )
  }

  if (status === 'Pending') {
    return (
      <span className="text-muted-foreground text-xs">{t('modelDownload.progress.waiting')}</span>
    )
  }

  const hasTotal = sizeBytes > 0
  const percent = hasTotal ? Math.min(100, Math.round((downloadedBytes / sizeBytes) * 100)) : 0

  return (
    <div className={cn('w-full min-w-[140px] space-y-1', className)}>
      {hasTotal ? (
        <Progress value={percent} />
      ) : (
        <Progress
          value={100}
          className={cn(status === 'Downloading' && 'animate-pulse', 'opacity-50')}
        />
      )}
      <p className="text-muted-foreground text-xs">
        {hasTotal
          ? `${percent}% · ${formatFileSize(downloadedBytes)} / ${formatFileSize(sizeBytes)}`
          : downloadedBytes > 0
            ? t('modelDownload.progress.downloaded', {
                size: formatFileSize(downloadedBytes),
              })
            : '--'}
        {status === 'Downloading' && downloadSpeed ? ` · ${downloadSpeed}` : ''}
      </p>
    </div>
  )
}

export default ModelDownloadProgress
