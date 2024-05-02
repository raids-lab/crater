import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FileSelect } from "@/components/custom/FileSelect";
import { useState } from "react";
import { Button } from "@/components/ui/button";

interface DialogProps extends React.HTMLAttributes<HTMLDivElement> {
  handleSubpathInfo: (info: string) => void;
  index: number;
}

// 修改FileSelect组件初始化File数组为顶层目录
export function FileSelectDialog({ handleSubpathInfo, index }: DialogProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [subpath, setSubpath] = useState("");
  const handleSubpath = (info: string) => {
    setSubpath(info); // 设置状态为false以关闭对话框
  };
  const handleDialogClose = () => {
    setIsDialogOpen(false); // 设置状态为false以关闭对话框
  };

  return (
    <div className="w-[120px]">
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className=" h-[67vh] w-[90vw] max-w-screen-sm">
          <DialogHeader className="h-[10px] w-[90vw] max-w-screen-sm">
            <DialogTitle>选择项目文件夹</DialogTitle>
          </DialogHeader>
          <FileSelect
            id={`FileSelect.args.${index}.subPath`}
            onClose={handleDialogClose}
            handleSubpathInfo={handleSubpathInfo}
            handleSubpath={handleSubpath}
          />
        </DialogContent>
      </Dialog>
      <Button
        id={`button.args.${index}.subPath`}
        type="button"
        variant="outline"
        role="combobox"
        onClick={() => {
          setIsDialogOpen(true);
        }} // 点击按钮打开对话框
        className=""
      >
        {subpath === "" ? "选择目录" : subpath}
      </Button>
    </div>
  );
}
