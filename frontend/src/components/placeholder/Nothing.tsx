import { GridIcon } from "lucide-react";

const Nothing = ({ title }: { title?: string }) => {
  return (
    <div className="flex h-[calc(100vh-_304px)] w-full items-center justify-center text-center text-muted-foreground/85 hover:bg-transparent">
      <div className="flex flex-col items-center justify-center">
        <div className="mb-4 rounded-full bg-muted p-3">
          <GridIcon className="h-6 w-6" />
        </div>
        <p className="select-none text-sm">{title ?? "暂无数据"}</p>
      </div>
    </div>
  );
};

export default Nothing;
