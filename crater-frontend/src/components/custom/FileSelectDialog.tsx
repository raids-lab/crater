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
import { Tree, TreeDataItem } from "@/components/custom/LazyFileTree";
import { ChevronsUpDown, FileDigit, Folder } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

// 修改FileSelect组件初始化File数组为顶层目录
export const FileSelectDialog = ({
  value,
  handleSubmit,
}: {
  value?: string;
  handleSubmit: (path: TreeDataItem) => void;
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
              "w-full justify-between text-ellipsis whitespace-nowrap pl-3 pr-4 font-normal",
              !value && "text-muted-foreground",
            )}
          >
            {!isDialogOpen && value ? value : "选择文件或文件夹"}
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>选择挂载数据</DialogTitle>
            <DialogDescription>支持选择单个文件或文件夹</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-4">
            <div className="relative flex h-80 flex-col gap-2 rounded-md border border-input shadow-sm">
              <Tree
                className="h-full w-full flex-shrink-0"
                onSelectChange={(item) => {
                  setContent(item);
                }}
                folderIcon={Folder}
                itemIcon={FileDigit}
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
