import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import { ReactNode } from "react";

const TipBadge = ({
  className,
  title,
}: {
  className?: string;
  title?: ReactNode;
}) => {
  return (
    <Badge
      className={cn(
        "rounded-full px-2 py-0 shadow-none select-none",
        "bg-highlight-orange/15 text-highlight-orange",
        className,
      )}
    >
      {title || "WIP"}
    </Badge>
  );
};

export default TipBadge;
