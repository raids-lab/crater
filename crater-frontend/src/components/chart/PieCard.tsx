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
    <Card className={cn("relative", className)}>
      {isLoading && (
        <div className="absolute bottom-0 left-0 right-0 top-0 z-10 flex items-center justify-center">
          <LoadingCircleIcon />
        </div>
      )}
      <CardHeader className="pb-3">
        <CardTitle className="flex flex-row items-center">
          <TooltipProvider delayDuration={100}>
            <Tooltip>
              <TooltipTrigger className="flex cursor-help flex-row items-center">
                <props.icon className="mr-1.5 size-5 text-muted-foreground" />
                {cardTitle}
              </TooltipTrigger>
              <TooltipContent className="border bg-background text-foreground">
                {cardDescription}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </CardTitle>
      </CardHeader>
      <div className="flex h-44 items-center justify-center px-2">
        {children}
      </div>
    </Card>
  );
};

export default PieCard;
