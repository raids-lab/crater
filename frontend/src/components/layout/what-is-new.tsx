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
const CURRENT_VERSION = '1.0.0'

// This would be your markdown content
const WHATS_NEW_CONTENT = `
## 版本 1.0.0

> 从 2023 年 7 月第一个前端提交开始，到今天，Crater 已走过近两年九个月。
> 这段时间里，我们累计完成了近千次代码提交、300 余次功能迭代。
> 在这个春天，Crater 终于迎来了 1.0.0 正式版。

### 本次更新

- 作业体系更完整：支持 Jupyter、WebIDE、自定义任务、PyTorch DDP、TensorFlow PS 等多种作业类型
- 开发流程更顺手：支持作业模板、作业克隆、日志查看、事件追踪，以及终止原因与资源明细展示
- 镜像能力更成熟：支持 Dockerfile、Envd、Pip / Apt 等多种构建方式，也支持镜像上传、克隆、分享与标签筛选
- 数据与文件管理更清晰：支持数据集、模型、共享块、文件系统管理，以及共享协作与权限控制
- 平台视图与管理能力更完善：支持 GPU、网络、空闲资源监控，支持节点详情、平台统计与 GPU 分析，并提供用户、配额、审批单、Cron Job、计费点数等管理功能

### 接下来

- 更好的国产化与异构硬件支持
- 更智能的运维与平台治理能力
- 面向 Harness 时代工作流的 CLI
- 持续升级存储系统的稳定性与体验
- 更加公平可靠的调度系统
- 以及更多正在路上

感谢一路以来的使用、反馈与催促。

欢迎在仓库留下 Star 或提交 Issue，和我们一起把 Crater 做得更好🥳
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
