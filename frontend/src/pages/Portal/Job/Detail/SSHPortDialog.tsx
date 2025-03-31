"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "@/components/ui/dialog";
import { CopyableCommand } from "@/components/codeblock/CopyableCommand";
import { Terminal } from "lucide-react";
import DocsButton from "@/components/button/DocsButton";

interface SSHPortDialogProps {
  hostIP: string;
  nodePort: number;
  userName: string;
  withButton?: boolean; // 是否展示按钮
  open?: boolean; // 外部控制打开
  onOpenChange?: (open: boolean) => void; // 控制变化回调
}

export function SSHPortDialog({
  hostIP,
  nodePort,
  userName,
  withButton = true,
  open,
  onOpenChange,
}: SSHPortDialogProps) {
  const [internalOpen, setInternalOpen] = useState(false);

  const isControlled = open !== undefined;
  const dialogOpen = isControlled ? open : internalOpen;
  const handleOpenChange = isControlled ? onOpenChange : setInternalOpen;

  return (
    <Dialog open={dialogOpen} onOpenChange={handleOpenChange}>
      {withButton && (
        <DialogTrigger asChild>
          <Button variant="secondary" title="打开 SSH 端口">
            <Terminal className="size-4" />
            SSH 端口
          </Button>
        </DialogTrigger>
      )}
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>SSH 端口信息</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col space-y-4">
          <CopyableCommand
            label="Terminal"
            command={`ssh ${userName}@${hostIP} -p ${nodePort}`}
          />
          <CopyableCommand
            label="VSCode"
            command={`${userName}@${hostIP}:${nodePort}`}
          />
        </div>
        <DialogFooter className="mt-4">
          <DocsButton title="帮助文档" url="toolbox/ssh/ssh-new" />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
