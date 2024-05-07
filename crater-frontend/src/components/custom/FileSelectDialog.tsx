import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FileSelect } from "@/components/custom/FileSelect";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { DialogTrigger } from "@radix-ui/react-dialog";

interface DialogProps extends React.HTMLAttributes<HTMLDivElement> {
  handleSubpathInfo: (info: string) => void;
}

// 修改FileSelect组件初始化File数组为顶层目录
export function FileSelectDialog({ handleSubpathInfo }: DialogProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [subpath, setSubpath] = useState("");
  const handleSubpath = (info: string) => {
    setSubpath(info);
  };

  return (
    <div>
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogTrigger asChild>
          <Button type="button" variant="outline">
            {subpath === "" ? "选择目录" : subpath}
          </Button>
        </DialogTrigger>
        <DialogContent className="h-[67vh] w-[90vw] max-w-screen-sm">
          <DialogHeader className="h-[10px] w-[90vw] max-w-screen-sm">
            <DialogTitle>选择项目文件夹</DialogTitle>
          </DialogHeader>
          <FileSelect
            onClose={() => setIsDialogOpen(false)}
            handleSubpathInfo={handleSubpathInfo}
            handleSubpath={handleSubpath}
          />
        </DialogContent>
      </Dialog>
    </div>
  );
}
