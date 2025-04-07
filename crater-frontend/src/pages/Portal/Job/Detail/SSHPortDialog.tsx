import { useState } from "react";
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
import { apiOpenSSH, SSHInfo } from "@/services/api/vcjob";
import { useMutation } from "@tanstack/react-query";
import LoadableButton from "@/components/custom/LoadableButton";
import { getErrorCode } from "@/services/axios";
import { ERROR_SERVICE_SSHD_NOT_FOUND } from "@/services/error_code";
import { toast } from "sonner";

interface SSHPortDialogProps {
  jobName: string;
  userName: string;
  withButton?: boolean; // 是否展示按钮
  open?: boolean; // 外部控制打开
  onOpenChange?: (open: boolean) => void; // 控制变化回调
}

export function SSHPortDialog({
  jobName,
  userName,
  withButton = true,
  open,
  onOpenChange,
}: SSHPortDialogProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const [sshInfo, setSSHInfo] = useState<SSHInfo | null>(null);

  const isControlled = open !== undefined;
  const dialogOpen = isControlled ? open : internalOpen;
  const handleOpenChange = isControlled ? onOpenChange : setInternalOpen;

  const { mutate: openSSH, isPending } = useMutation({
    mutationFn: () => apiOpenSSH(jobName),
    onSuccess: (response) => {
      const data = response.data.data;
      setSSHInfo(data);
      handleOpenChange?.(true);
    },
    onError: (error) => {
      const [errorCode] = getErrorCode(error);
      if (errorCode === ERROR_SERVICE_SSHD_NOT_FOUND) {
        toast.error("SSHD 服务未发现，请阅读 SSH 相关文档。");
      }
    },
  });

  return (
    <Dialog open={dialogOpen} onOpenChange={handleOpenChange}>
      {withButton && (
        <DialogTrigger asChild>
          <LoadableButton
            variant="secondary"
            title="打开 SSH 端口"
            className="cursor-pointer"
            onClick={(e) => {
              e.preventDefault();
              if (!sshInfo) {
                openSSH();
              } else {
                handleOpenChange?.(true);
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
        {sshInfo ? (
          <div className="flex flex-col space-y-4">
            <CopyableCommand
              label="Terminal"
              command={`ssh ${userName}@${sshInfo.ip} -p ${sshInfo.port}`}
            />
            <CopyableCommand
              label="VSCode"
              command={`${userName}@${sshInfo.ip}:${sshInfo.port}`}
            />
          </div>
        ) : (
          <div className="py-4 text-center">
            <p>正在获取 SSH 连接信息...</p>
          </div>
        )}
        <DialogFooter className="mt-4">
          <DocsButton title="帮助文档" url="toolbox/ssh/ssh-new" />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
