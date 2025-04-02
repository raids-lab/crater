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
import { Trash2, X, AlertTriangle } from "lucide-react";

interface DeleteDialogProps {
  imageLinks: string[];
  onDeleteImageList: () => void;
}

export const DeleteDialog: FC<DeleteDialogProps> = ({
  imageLinks,
  onDeleteImageList,
}) => {
  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2 text-xl">
          <Trash2 className="text-destructive h-5 w-5" />
          <span>删除镜像</span>
        </AlertDialogTitle>
      </AlertDialogHeader>

      <Separator className="my-3" />

      <AlertDialogDescription className="space-y-4 pt-2">
        <div className="border-destructive/20 bg-destructive/5 rounded-md border px-4 py-3">
          <div className="flex items-start gap-3">
            <AlertTriangle className="text-destructive mt-0.5 h-5 w-5 shrink-0" />
            <div>
              <p className="text-destructive font-medium">将删除以下镜像</p>
              <p className="text-muted-foreground mt-1 text-sm">
                {"『" + imageLinks.join("』,『") + "』"}
              </p>
            </div>
          </div>
        </div>
      </AlertDialogDescription>

      <AlertDialogFooter>
        <AlertDialogCancel>
          <X />
          取消
        </AlertDialogCancel>

        <AlertDialogAction variant="destructive" onClick={onDeleteImageList}>
          <Trash2 />
          确认删除
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
};
