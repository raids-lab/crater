import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";

const TipBadge = ({
  className,
  title,
}: {
  className?: string;
  title?: string;
}) => {
  return (
    <Badge
      className={cn(
        "select-none rounded-full bg-orange-600/15 px-2 py-0 uppercase text-orange-600 shadow-none hover:bg-orange-600/25",
        className,
      )}
    >
      {title || "WIP"}
    </Badge>
  );
};

export default TipBadge;
