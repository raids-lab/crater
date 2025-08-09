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
// i18n-processed-v1.1.0
// Modified code
import { Globe, Info, MonitorOff } from 'lucide-react'
import { type FC } from 'react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { cn } from '@/lib/utils'

type BasicIframeProps = React.IframeHTMLAttributes<HTMLIFrameElement>

export const BasicIframe: FC<BasicIframeProps> = ({
  src,
  className = '',
  title,
  ...iframeProps
}) => {
  const { t } = useTranslation()

  // 检查是否为开发模式
  const isDevelopment = import.meta.env.MODE === 'development'

  // 开发模式下显示占位图
  if (isDevelopment) {
    return (
      <div className={cn('relative h-full w-full overflow-hidden rounded-xl', className)}>
        {/* 毛玻璃效果层 */}
        <div className="bg-sidebar absolute inset-0" />

        {/* 居中内容卡片 */}
        <div className="absolute inset-0 flex items-center justify-center p-8">
          <Card className="bg-background/80 border-border/50 mx-auto w-full max-w-md shadow-2xl backdrop-blur-md">
            <CardHeader className="pb-3 text-center">
              <div className="mb-3 flex justify-center">
                <div className="bg-primary/10 rounded-full p-3">
                  <MonitorOff className="text-highlight-orange h-8 w-8" />
                </div>
              </div>
              <CardTitle className="text-xl font-semibold">
                {t('basicIframe.developmentMode.title')}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-2">
                <Globe className="text-highlight-orange h-4 w-4" />
                <span className="text-sm">{t('basicIframe.developmentMode.crossOrigin')}</span>
              </div>

              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Info className="text-highlight-orange h-4 w-4" />
                  <span className="text-sm font-medium">
                    {t('basicIframe.info.source')} {title}
                  </span>
                </div>
                <p className="bg-muted/50 text-muted-foreground rounded p-2 font-mono text-xs break-all">
                  {src}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  // 生产模式或禁用占位图时显示正常的 iframe
  return (
    <iframe
      src={src}
      title={title}
      className={cn('h-full w-full rounded-xl border-none', className)}
      {...iframeProps}
    />
  )
}

export default BasicIframe
