// i18n-processed-v1.1.0 (no translatable strings)
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
        <TooltipContent>{tooltipContent}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

TooltipButton.displayName = "TooltipButton";
export default TooltipButton;
