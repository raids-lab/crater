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
import { QrCode, RocketIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useLocalStorage } from 'usehooks-ts'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'

import { MarkdownRenderer } from '@/components/form/markdown-renderer'

// Current app version - update this when you release new features
const CURRENT_VERSION = '0.0.0'

// This would be your markdown content
const WHATS_NEW_CONTENT = `
## 版本 0.0.0

> 开源 & 0.0.1 版本的发布已经不远了 🥳

### 新功能
- 新增「共享文件」功能，您可以把自己的 \`Home\` 目录分享给其他用户读写了 
- 提供了 CUDA 11~12 的基础镜像，快从 Slurm Conda 环境迁移吧，后者真的很慢
- 分布式作业支持 RDMA（基于 InfiniBand），和 TCP 连接说再见
- 自定义任务可以用普通用户而非 \`root\` 用户提交了，但您还需要阅读一下提交页面的文档，我们在努力优化 💪

### 问题修复
- 分布式作业的「外部访问」功能现在正常了
- 邮件通知的日期显示正常了，样式也更好看了

## 即将到来
- 镜像按照标签搜索，Python、CUDA、Jupyter... 想要什么搜什么！
- 数据缓存优化，多次访问同一数据集时极大提速，对标企业级体验！
- 作业锁定的工单系统，不用再在群里 @ 管理员，小需求快捷通过
`

interface WhatsNewDialogProps {
  // You can pass a custom version if needed
  version?: string
}

export function WhatsNewDialog({ version = CURRENT_VERSION }: WhatsNewDialogProps) {
  const [lastConfirmedVersion, setLastConfirmedVersion] = useLocalStorage<string>(
    'app-last-confirmed-version',
    ''
  )
  const [open, setOpen] = useState(false)

  useEffect(() => {
    // Check if the current version is different from the last confirmed version
    if (lastConfirmedVersion !== version) {
      setOpen(true)
    }
  }, [lastConfirmedVersion, version])

  const handleConfirm = () => {
    setLastConfirmedVersion(version)
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>最近更新内容 v{version}</DialogTitle>
          <DialogDescription>
            {/* Check out the latest updates and improvements we&apos;ve made to
            enhance your experience. */}
            查看我们最近的更新和改进，以提升您的使用体验
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="mt-4 max-h-[300px]">
          <MarkdownRenderer>{WHATS_NEW_CONTENT}</MarkdownRenderer>
        </ScrollArea>

        <div className="bg-muted/50 mt-6 rounded-lg border p-4">
          <div className="flex items-start gap-4">
            <div className="bg-primary/10 rounded-lg p-2">
              <QrCode className="text-primary h-8 w-8" />
            </div>
            <div>
              <h4 className="text-sm font-medium">加入用户交流群</h4>
              <p className="text-muted-foreground mt-1 text-sm">
                由于微信群聊已满 200 人，联系您身边的老师 /
                同学以加入我们的用户交流群，获取平台最新动态和技术支持。
              </p>
              {/* <Button variant="outline" size="sm" className="mt-2">
                加入交流群
              </Button> */}
            </div>
          </div>
        </div>

        <DialogFooter className="mt-4">
          <Button onClick={handleConfirm}>
            确认，开始使用
            <RocketIcon />
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
