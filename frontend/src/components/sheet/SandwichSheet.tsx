import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
  SheetTrigger,
} from "@/components/ui/sheet";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import useResizeObserver from "use-resize-observer";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ReactNode } from "react";
import { HelpCircleIcon } from "lucide-react";

export interface SandwichSheetProps {
  isOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  title: ReactNode;
  description?: ReactNode;
  className?: string;
}

interface SheetProps extends SandwichSheetProps {
  isOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  title: ReactNode;
  description?: ReactNode;
  className?: string;
  children: ReactNode;
  trigger?: ReactNode;
}

const SandwichSheet = ({
  isOpen,
  onOpenChange,
  title,
  description,
  className,
  children,
  trigger,
}: SheetProps) => {
  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      {trigger && <SheetTrigger asChild>{trigger}</SheetTrigger>}
      <SheetContent className={cn("overflow-hidden p-0", className)}>
        <div className="relative -z-10 h-screen">
          <SheetHeader className="h-[72px] pt-6 pb-4 pl-6">
            <SheetTitle className="flex flex-row items-center">
              {title}
              {description && (
                <TooltipProvider delayDuration={100}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <HelpCircleIcon className="text-muted-foreground ml-1 size-4 hover:cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent>{description}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </SheetTitle>
          </SheetHeader>
          {children}
        </div>
      </SheetContent>
    </Sheet>
  );
};

export const SandwichLayout = ({
  children,
  footer,
}: {
  children: ReactNode;
  footer?: ReactNode;
}) => {
  const { ref: refRoot, width, height } = useResizeObserver();
  return (
    <div className="h-screen">
      <div
        className={cn({
          "h-[calc(100vh_-_156px)]": footer,
          "h-[calc(100vh_-_72px)]": !footer,
        })}
        ref={refRoot}
      >
        <ScrollArea style={{ width, height }}>
          <div className="z-50 space-y-4 px-6">{children}</div>
        </ScrollArea>
      </div>
      {footer && (
        <SheetFooter className="absolute right-0 bottom-0 left-0 h-auto flex-row justify-end gap-2 p-6">
          {footer}
        </SheetFooter>
      )}
    </div>
  );
};

export default SandwichSheet;
