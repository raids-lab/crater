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
import { QuestionMarkCircledIcon } from "@radix-ui/react-icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ReactNode } from "react";

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
  footer?: ReactNode;
  trigger?: ReactNode;
}

const SandwichSheet = ({
  isOpen,
  onOpenChange,
  title,
  description,
  className,
  children,
  footer,
  trigger,
}: SheetProps) => {
  const { ref: refRoot, width, height } = useResizeObserver();

  return (
    <Sheet
      open={trigger ? undefined : isOpen}
      onOpenChange={trigger ? undefined : onOpenChange}
    >
      {trigger && <SheetTrigger asChild>{trigger}</SheetTrigger>}
      <SheetContent className={cn("p-0", className)}>
        <div className="relative -z-10 h-screen">
          <SheetHeader className="h-[72px] pb-4 pl-6 pt-6">
            <SheetTitle className="flex flex-row items-center">
              {title}
              {description && (
                <TooltipProvider delayDuration={100}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <QuestionMarkCircledIcon className="ml-1 size-4 text-muted-foreground hover:cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent className="border bg-background text-foreground">
                      {description}
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </SheetTitle>
          </SheetHeader>
          <div
            className={cn({
              "h-[calc(100vh-_156px)]": footer,
              "h-[calc(100vh-_72px)]": !footer,
            })}
            ref={refRoot}
          >
            <ScrollArea style={{ width, height }} className="">
              {children}
            </ScrollArea>
          </div>
          {footer && (
            <SheetFooter className="absolute bottom-0 left-0 right-0 gap-2 p-6">
              {footer}
            </SheetFooter>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default SandwichSheet;
