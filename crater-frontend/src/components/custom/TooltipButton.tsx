import React from "react";
import { buttonVariants, Button } from "../ui/button";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { VariantProps } from "class-variance-authority";

type TooltipButtonProps = React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    tooltipContent: string;
  };

const TooltipButton = function LoadableButton({
  ref,
  className,
  variant,
  size,
  tooltipContent,
  children,
  ...props
}: TooltipButtonProps & {
  ref?: React.RefObject<HTMLButtonElement>;
}) {
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
};

TooltipButton.displayName = "TooltipButton";
export default TooltipButton;
