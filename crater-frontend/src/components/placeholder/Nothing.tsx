import { cn } from "@/lib/utils";
import { GridIcon } from "lucide-react";

export const NothingCore = ({
  title,
  className,
}: {
  title?: string;
  className?: string;
}) => {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-4",
        className,
      )}
    >
      <div className="bg-muted rounded-full p-3">
        <GridIcon className="text-muted-foreground h-6 w-6" />
      </div>
      <p className="text-muted-foreground text-sm select-none">
        {title ?? "暂无数据"}
      </p>
    </div>
  );
};

const Nothing = ({ title }: { title?: string }) => {
  return (
    <div className="text-muted-foreground/85 flex h-[calc(100vh-_304px)] w-full items-center justify-center text-center hover:bg-transparent">
      <NothingCore title={title} />
    </div>
  );
};

export default Nothing;
