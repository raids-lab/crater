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
        "rounded-full px-2 py-0 uppercase shadow-none select-none",
        "bg-orange-600/15 text-orange-600 hover:bg-orange-600/25",
        className,
      )}
    >
      {title || "WIP"}
    </Badge>
  );
};

export default TipBadge;
