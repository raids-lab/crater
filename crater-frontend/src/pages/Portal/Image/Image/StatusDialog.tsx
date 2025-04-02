import { type FC } from "react";
import { Separator } from "@/components/ui/separator";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui-custom/alert-dialog";
import { Check, Globe, Lock, X } from "lucide-react";
import VisibilityBadge, {
  Visibility,
} from "@/components/badge/VisibilityBadge";

interface StatusDialogProps {
  imageLink: string;
  isPublic: boolean;
  onChange: () => void;
}

export const StatusDialog: FC<StatusDialogProps> = ({
  imageLink,
  isPublic,
  onChange,
}) => {
  const currentVisibility = isPublic ? Visibility.Public : Visibility.Private;
  const newVisibility = isPublic ? Visibility.Private : Visibility.Public;

  // Choose icon based on the new status
  const StatusIcon = isPublic ? Lock : Globe;
  const statusColor = isPublic ? "text-amber-600" : "text-green-600";
  const bgColor = isPublic ? "bg-amber-50" : "bg-green-50";
  const darkBgColor = isPublic
    ? "dark:bg-amber-950/30"
    : "dark:bg-green-950/30";
  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2 text-xl">
          <StatusIcon className={`h-5 w-5 ${statusColor}`} />
          <span>更新镜像访问权限</span>
        </AlertDialogTitle>
      </AlertDialogHeader>

      <Separator className="my-3" />

      <AlertDialogDescription className="space-y-4 pt-2">
        <div className="bg-muted/50 rounded-md px-4 py-3">
          <p className="text-muted-foreground text-sm">镜像链接</p>
          <p className="mt-1 font-medium break-all">『{imageLink}』</p>
        </div>

        <div className={`rounded-md ${bgColor} ${darkBgColor} p-4`}>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium">状态变更</p>
              <div className="flex items-center gap-6">
                <div className="flex items-center gap-1.5">
                  <VisibilityBadge visibility={currentVisibility} />
                </div>
                <div className="text-muted-foreground">→</div>
                <div className="flex items-center gap-1.5">
                  <VisibilityBadge visibility={newVisibility} />
                </div>
              </div>
            </div>
            <StatusIcon className={`h-8 w-8 ${statusColor} opacity-20`} />
          </div>
        </div>

        <p className="text-muted-foreground text-sm">
          {isPublic
            ? "设为私有后，只有您可以访问此镜像。"
            : "设为公共后，任何人都可以通过链接访问此镜像。"}
        </p>
      </AlertDialogDescription>

      <AlertDialogFooter>
        <AlertDialogCancel>
          <X />
          取消
        </AlertDialogCancel>

        <AlertDialogAction
          className={`flex items-center gap-2 ${isPublic ? "bg-amber-600 hover:bg-amber-700" : "bg-green-600 hover:bg-green-700"}`}
          onClick={onChange}
        >
          <Check />
          确认
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
};
