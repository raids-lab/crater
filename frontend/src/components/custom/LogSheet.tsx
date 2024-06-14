import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import useResizeObserver from "use-resize-observer";

type LogSheetProps = React.HTMLAttributes<HTMLDivElement> & {
  title?: string;
  log?: string;
  side?: "bottom" | "left" | "right" | "top" | null | undefined;
};

const LogSheet = ({ title, log, side, children }: LogSheetProps) => {
  const { ref: refRoot, width, height } = useResizeObserver();

  return (
    <Sheet>
      <SheetTrigger asChild>{children}</SheetTrigger>
      {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
      <SheetContent className="flex flex-col gap-6 sm:max-w-3xl" side={side}>
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
        </SheetHeader>
        <Card
          className="h-[calc(100vh_-104px)] bg-gray-100 text-muted-foreground dark:border dark:bg-transparent"
          ref={refRoot}
        >
          <ScrollArea style={{ width, height }}>
            <CardHeader className="py-3"></CardHeader>
            <CardContent>
              <pre className="whitespace-pre-wrap text-sm">{log}</pre>
            </CardContent>
            <ScrollBar orientation="horizontal" />
          </ScrollArea>
        </Card>
      </SheetContent>
    </Sheet>
  );
};

export default LogSheet;
