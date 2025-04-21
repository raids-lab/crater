import { useMutation, useQuery } from "@tanstack/react-query";
import { type FC, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  apiGetHarborIP,
  apiUserGetCredential,
  ProjectCredentialResponse,
} from "@/services/api/imagepack";
import { toast } from "sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui-custom/alert-dialog";
import { Key, AlertTriangle, RefreshCw, SailboatIcon } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { CopyableCommand } from "@/components/codeblock/CopyableCommand";
import { Badge } from "@/components/ui/badge";

interface UserHarborCredentialsDialogProps {
  isDialogOpen: boolean;
  setIsDialogOpen: (open: boolean) => void;
}

export const UserHarborCredentialsDialog: FC<
  UserHarborCredentialsDialogProps
> = ({ isDialogOpen, setIsDialogOpen }) => {
  const [credentials, setCredentials] =
    useState<ProjectCredentialResponse | null>(null);
  const [showConfirmation, setShowConfirmation] = useState(false);
  const user = useAtomValue(globalUserInfo);
  const { mutate: getProjectCredential } = useMutation({
    mutationFn: () => apiUserGetCredential(),
    onSuccess: async (data) => {
      setCredentials(data.data.data);
      setShowConfirmation(false);
      toast.success("凭据已生成, 请保存您的密码！");
    },
  });
  const harborIP = useQuery({
    queryKey: ["harbor", "ip"],
    queryFn: () => apiGetHarborIP(),
    select: (res) => res.data.data,
  });
  const handleResetClick = () => {
    setShowConfirmation(true);
  };
  return (
    <>
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Key className="h-6 w-6 text-blue-500" />
              镜像仓库访问凭据
            </DialogTitle>
          </DialogHeader>

          <Alert className="my-3 border-yellow-500 bg-yellow-50 dark:border-yellow-500/50 dark:bg-yellow-950/30">
            <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
            <AlertDescription className="font-medium text-yellow-800 dark:text-yellow-300">
              请保存好您的用户名和密码，密码只会显示一次！
            </AlertDescription>
          </Alert>

          <Card className="border-blue-100 bg-blue-50/50 dark:border-blue-900/30 dark:bg-blue-950/20">
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center justify-between text-base font-medium">
                <span className="flex items-center gap-2">
                  <SailboatIcon className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                  仓库访问信息
                </span>
                <Badge
                  variant="outline"
                  className="bg-blue-100/50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
                >
                  Harbor Registry
                </Badge>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <CopyableCommand
                label="仓库地址"
                command={harborIP.data?.ip || "加载中..."}
                isLink={true}
              />
              <CopyableCommand label="项目名称" command={"user-" + user.name} />
              <CopyableCommand label="用户名" command={user.name} />
              <CopyableCommand
                label="密码"
                command={credentials ? credentials.password : "•".repeat(10)}
                isSensitive={!credentials}
              />
            </CardContent>
          </Card>

          <DialogFooter className="mt-4 sm:justify-between">
            <Button
              type="button"
              variant="outline"
              className="flex items-center gap-2"
              onClick={() => handleResetClick()}
            >
              <RefreshCw className="h-4 w-4" />
              重置密码
            </Button>
            <Button
              type="button"
              variant="default"
              onClick={() => setIsDialogOpen(false)}
            >
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <AlertDialog open={showConfirmation} onOpenChange={setShowConfirmation}>
        <AlertDialogContent className="max-w-[350px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确认重置密码</AlertDialogTitle>
            <AlertDialogDescription>
              您确定要重置密码吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => getProjectCredential()}
              className="bg-red-500 hover:bg-red-600"
            >
              确认重置
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};
