import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";

const WIPBadge = ({ className }: { className?: string }) => {
  return (
    <Badge
      className={cn(
        "rounded-full bg-orange-600/15 px-1 py-0 text-orange-600 shadow-none hover:bg-orange-600/25",
        className,
      )}
    >
      WIP
    </Badge>
  );
};

export default WIPBadge;
