// https://github.com/shadcn-ui/ui/issues/355
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { DialogTrigger } from "@radix-ui/react-dialog";
import { Tree, TreeDataItem } from "@/components/file/LazyFileTree";
import { ChevronsUpDown } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

// 修改FileSelect组件初始化File数组为顶层目录
export const FileSelectDialog = ({
  value,
  handleSubmit,
  disabled,
}: {
  value?: string;
  handleSubmit: (path: TreeDataItem) => void;
  disabled?: boolean;
}) => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [content, setContent] = useState<TreeDataItem | undefined>();

  const handleSelect = () => {
    if (content) {
      handleSubmit(content);
      toast.info(`选择了${content.id}`);
      setIsDialogOpen(false);
    } else {
      toast.warning("请选择一个文件或文件夹");
    }
  };

  return (
    <div>
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogTrigger asChild>
          <Button
            variant="outline"
            role="file-select"
            className={cn(
              "focus:outline-primary w-full justify-between pr-4 pl-3 font-normal text-ellipsis whitespace-nowrap",
              !value && "text-muted-foreground",
            )}
            disabled={disabled}
          >
            {!isDialogOpen && value ? value : "选择文件"}
            <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>选择挂载数据</DialogTitle>
            <DialogDescription>支持选择单个文件或文件夹</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-4">
            <div className="border-input relative flex h-80 flex-col gap-2 rounded-md border shadow-xs">
              <Tree
                className="h-full w-full shrink-0"
                onSelectChange={(item) => {
                  setContent(item);
                }}
              />
            </div>
          </div>
          <DialogFooter className="gap-2 sm:justify-end sm:space-x-0">
            <Button type="button" className="w-full" onClick={handleSelect}>
              确认选择 {content?.name}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
