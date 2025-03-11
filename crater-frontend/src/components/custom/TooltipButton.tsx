import React from "react";
import { ButtonProps, buttonVariants, Button } from "../ui/button";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface TooltipButtonProps extends ButtonProps {
  tooltipContent: string;
}

const TooltipButton = React.forwardRef<HTMLButtonElement, TooltipButtonProps>(
  function LoadableButton(
    { className, variant, size, tooltipContent, children, ...props },
    ref,
  ) {
    return (
      <TooltipProvider delayDuration={100}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant={variant}
              size={size}
              className={cn(buttonVariants({ variant, size, className }))}
              ref={ref}
              {...props}
            >
              {children}
            </Button>
          </TooltipTrigger>
          <TooltipContent className="bg-background text-foreground max-w-44 border">
            {tooltipContent}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  },
);

TooltipButton.displayName = "TooltipButton";
export default TooltipButton;
