import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Card } from "@/components/ui/card";
import BaseCodeBlock from "./BaseCodeBlock";
import useResizeObserver from "use-resize-observer";
import { useCopyToClipboard } from "usehooks-ts";
import { Button } from "../ui/button";
import { CopyCheckIcon, CopyIcon } from "lucide-react";
import { toast } from "sonner";

type CodeSheetProps = React.HTMLAttributes<HTMLDivElement> & {
  title?: string;
  code?: string;
  language?: string;
  side?: "bottom" | "left" | "right" | "top" | null | undefined;
};

const CodeSheet = ({
  title = "Code",
  code = "",
  language = "yaml",
  side,
  children = null,
}: CodeSheetProps) => {
  const { ref: refRoot, width, height } = useResizeObserver();
  const [copiedText, copy] = useCopyToClipboard();

  const copyCode = () => {
    // Logic to copy `code`
    copy(code ?? "")
      .then(() => {
        toast.success("已复制到剪贴板");
      })
      .catch(() => {
        toast.error("复制失败");
      });
  };

  return (
    <Sheet>
      <SheetTrigger asChild>{children}</SheetTrigger>
      {/* scroll in sheet: https://github.com/shadcn-ui/ui/issues/16 */}
      <SheetContent className="flex flex-col gap-6 sm:max-w-3xl" side={side}>
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
        </SheetHeader>
        <Card
          className="relative h-[calc(100vh_-104px)] w-full overflow-hidden bg-gray-100 text-muted-foreground dark:border dark:bg-transparent"
          ref={refRoot}
        >
          <ScrollArea style={{ width, height }}>
            {/* <pre className="whitespace-pre-wrap text-sm">{log}</pre> */}
            <BaseCodeBlock code={code ?? ""} language={language ?? ""} />
            <ScrollBar orientation="horizontal" />
          </ScrollArea>
          <Button
            className="absolute right-4 top-4 h-8 w-8"
            onClick={copyCode}
            variant="outline"
            size="icon"
          >
            {copiedText ? (
              <CopyCheckIcon className="h-4 w-4" />
            ) : (
              <CopyIcon className="h-4 w-4" />
            )}
          </Button>
        </Card>
      </SheetContent>
    </Sheet>
  );
};

export default CodeSheet;
