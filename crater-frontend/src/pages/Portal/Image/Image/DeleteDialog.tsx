import { type FC } from "react";
import {
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
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
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2 text-xl">
          <Trash2 className="text-destructive h-5 w-5" />
          <span>删除镜像</span>
        </DialogTitle>
      </DialogHeader>

      <DialogDescription className="space-y-4 pt-2">
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
      </DialogDescription>

      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline" className="flex items-center gap-2">
            <X className="size-4" />
            取消
          </Button>
        </DialogClose>

        <Button
          variant="destructive"
          onClick={onDeleteImageList}
          className="flex items-center gap-2"
        >
          <Trash2 className="size-4" />
          确认删除
        </Button>
      </DialogFooter>
    </>
  );
};
