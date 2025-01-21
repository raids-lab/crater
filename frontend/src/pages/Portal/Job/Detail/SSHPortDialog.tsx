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
import { BookOpenIcon, Terminal } from "lucide-react";

interface SSHPortDialogProps {
  hostIP: string;
  nodePort: number;
  userName: string;
}

export function SSHPortDialog({
  hostIP,
  nodePort,
  userName,
}: SSHPortDialogProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="secondary" title="打开 SSH 端口">
          <Terminal className="size-4" />
          SSH 端口
        </Button>
      </DialogTrigger>
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
          <Button
            variant="secondary"
            onClick={() =>
              window.open(
                `https://${import.meta.env.VITE_HOST}/website/docs/toolbox/external-access/nodeport-rule`,
              )
            }
          >
            <BookOpenIcon />
            帮助文档
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
