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
import { useMutation } from '@tanstack/react-query'
import { Terminal } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

import DocsButton from '@/components/button/docs-button'
import LoadableButton from '@/components/button/loadable-button'
import { CopyableCommand } from '@/components/codeblock/copyable-command'

import { SSHInfo, apiOpenSSH } from '@/services/api/vcjob'
import { getErrorCode } from '@/services/client'
import { ERROR_SERVICE_SSHD_NOT_FOUND } from '@/services/error_code'

interface SSHPortDialogProps {
  jobName: string
  userName: string
  withButton?: boolean // 是否展示按钮
  open?: boolean // 外部控制打开
  onOpenChange?: (open: boolean) => void // 控制变化回调
}

export function SSHPortDialog({
  jobName,
  userName,
  withButton = true,
  open,
  onOpenChange,
}: SSHPortDialogProps) {
  const [internalOpen, setInternalOpen] = useState(false)
  const [sshInfo, setSSHInfo] = useState<SSHInfo | null>(null)

  const isControlled = open !== undefined
  const dialogOpen = isControlled ? open : internalOpen
  const handleOpenChange = isControlled ? onOpenChange : setInternalOpen

  const { mutate: openSSH, isPending } = useMutation({
    mutationFn: () => apiOpenSSH(jobName),
    onSuccess: (response) => {
      const data = response.data
      setSSHInfo(data)
      handleOpenChange?.(true)
    },
    onError: async (error) => {
      const [errorCode] = await getErrorCode(error)
      if (errorCode === ERROR_SERVICE_SSHD_NOT_FOUND) {
        toast.error(<div className="flex flex-row items-center">未检测到 SSHD 服务</div>)
      }
    },
  })

  return (
    <Dialog open={dialogOpen} onOpenChange={handleOpenChange}>
      {withButton && (
        <DialogTrigger asChild>
          <LoadableButton
            variant="secondary"
            title="打开 SSH 端口"
            className="cursor-pointer"
            onClick={(e) => {
              e.preventDefault()
              if (!sshInfo) {
                openSSH()
              } else {
                handleOpenChange?.(true)
              }
            }}
            isLoading={isPending}
            isLoadingText="SSH 连接"
          >
            <Terminal className="size-4" />
            SSH 连接
          </LoadableButton>
        </DialogTrigger>
      )}
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>SSH 连接</DialogTitle>
        </DialogHeader>
        {sshInfo && (
          <div className="flex flex-col space-y-4">
            <CopyableCommand
              label="通过 Terminal 连接"
              command={`ssh ${userName}@${sshInfo.ip} -p ${sshInfo.port}`}
            />
            <CopyableCommand
              label="通过 VSCode 连接"
              command={`${userName}@${sshInfo.ip}:${sshInfo.port}`}
            />
          </div>
        )}
        <DialogFooter className="mt-2">
          <DocsButton title="帮助文档" url="toolbox/ssh/ssh-func" />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
