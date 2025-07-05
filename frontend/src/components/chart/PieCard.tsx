import {
  TooltipProvider,
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { LucideIcon } from "lucide-react";
import LoadingCircleIcon from "../icon/LoadingCircleIcon";
import { Card, CardHeader, CardTitle } from "../ui/card";
import { cn } from "@/lib/utils";

interface PieCardProps extends React.HTMLAttributes<HTMLDivElement> {
  icon: LucideIcon;
  cardTitle: string;
  cardDescription: string;
  isLoading?: boolean;
}

const PieCard = ({
  children,
  cardTitle,
  cardDescription,
  isLoading,
  className,
  ...props
}: PieCardProps) => {
  return (
    <Card className={cn("relative pb-0", className)}>
      {isLoading && (
        <div className="absolute top-0 right-0 bottom-0 left-0 z-10 flex items-center justify-center">
          <LoadingCircleIcon />
        </div>
      )}
      <CardHeader>
        <CardTitle className="flex flex-row items-center">
          <TooltipProvider delayDuration={100}>
            <Tooltip>
              <TooltipTrigger className="flex cursor-help flex-row items-center">
                <props.icon className="text-primary mr-1.5 size-5" />
                {cardTitle}
              </TooltipTrigger>
              <TooltipContent>{cardDescription}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </CardTitle>
      </CardHeader>
      <div className="relative h-52 overflow-hidden px-2">{children}</div>
    </Card>
  );
};

export default PieCard;
