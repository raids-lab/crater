import { type FC, useEffect, useState } from "react";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui-custom/alert-dialog";
import { Check, Pencil, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@radix-ui/react-label";

interface RenameDialogProps {
  imageDescription: string;
  onRename: (newDescription: string) => void;
}

export const RenameDialog: FC<RenameDialogProps> = ({
  imageDescription,
  onRename,
}) => {
  const initialDescription = "";
  const [newDescription, setNewDescription] = useState(initialDescription);
  const [isTouched, setIsTouched] = useState(false);
  useEffect(() => {
    setNewDescription(initialDescription || "");
    setIsTouched(false);
  }, [initialDescription]);

  const isValid = newDescription.trim().length > 0;
  const hasChanged = newDescription !== initialDescription;
  const canSubmit = isValid && hasChanged;
  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-1 text-xl">
          <Pencil className="text-primary h-5 w-5" />
          <span>更新镜像名称</span>
        </AlertDialogTitle>
      </AlertDialogHeader>

      <Separator className="my-3" />

      <AlertDialogDescription className="space-y-4 pt-2">
        <div className="bg-muted/50 rounded-md px-4 py-3">
          <p className="text-muted-foreground text-sm">当前名称</p>
          <p className="mt-1 font-medium">「{imageDescription}」</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="new-description" className="font-medium">
            新名称
          </Label>
          <Input
            id="new-description"
            type="text"
            value={newDescription}
            onChange={(e) => {
              setNewDescription(e.target.value);
              setIsTouched(true);
            }}
            placeholder="输入新的描述"
            className={`transition-all ${isTouched && !isValid ? "border-destructive ring-destructive/10" : ""}`}
            autoFocus
          />
          {isTouched && !isValid && (
            <p className="text-destructive text-xs">请输入有效的名称</p>
          )}
        </div>
      </AlertDialogDescription>

      <AlertDialogFooter className="mt-4 gap-2 sm:gap-0">
        <AlertDialogCancel asChild>
          <Button variant="outline" className="flex items-center gap-2">
            <X className="h-4 w-4" />
            取消
          </Button>
        </AlertDialogCancel>

        <AlertDialogAction asChild>
          <Button
            variant="default"
            className="flex items-center gap-2"
            onClick={() => onRename(newDescription)}
            disabled={!canSubmit}
          >
            <Check className="h-4 w-4" />
            确认
          </Button>
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
};
