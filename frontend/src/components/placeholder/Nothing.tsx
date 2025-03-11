import { GridIcon } from "lucide-react";

const Nothing = ({ title }: { title?: string }) => {
  return (
    <div className="text-muted-foreground/85 flex h-[calc(100vh-_304px)] w-full items-center justify-center text-center hover:bg-transparent">
      <div className="flex flex-col items-center justify-center">
        <div className="bg-muted mb-4 rounded-full p-3">
          <GridIcon className="h-6 w-6" />
        </div>
        <p className="text-sm select-none">{title ?? "暂无数据"}</p>
      </div>
    </div>
  );
};

export default Nothing;
