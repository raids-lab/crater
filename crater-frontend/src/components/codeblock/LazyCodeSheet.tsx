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
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";

type Side = "bottom" | "left" | "right" | "top" | null | undefined;
type LazyCodeSheetProps = React.HTMLAttributes<HTMLDivElement> & {
  title?: string;
  queryFn?: () => Promise<string>;
  language?: string;
  side?: Side;
};

const CodeSheetContent = ({
  title = "Code",
  queryFn,
  language = "yaml",
}: {
  title: string;
  language?: string;
  queryFn?: () => Promise<string>;
}) => {
  const [copiedText, copy] = useCopyToClipboard();

  const { data: code } = useQuery({
    queryKey: ["code", "sheet", title],
    queryFn,
  });

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

  useEffect(() => {
    toast.info("CodeSheetContent mounted");
    return () => {
      toast.info("CodeSheetContent unmounted");
    };
  }, []);

  const { ref: refRoot, width, height } = useResizeObserver();
  return (
    <>
      <SheetHeader>
        <SheetTitle>{title}</SheetTitle>
      </SheetHeader>
      <Card
        ref={refRoot}
        className="text-muted-foreground relative h-[calc(100vh_-104px)] w-full overflow-hidden bg-gray-100 dark:border dark:bg-transparent"
      >
        <ScrollArea style={{ width, height }}>
          <BaseCodeBlock code={code ?? ""} language={language} />
          <ScrollBar orientation="horizontal" />
        </ScrollArea>
        <Button
          className="absolute top-4 right-4 h-8 w-8"
          onClick={copyCode}
          variant="outline"
          size="icon"
        >
          {copiedText ? (
            <CopyCheckIcon className="size-4" />
          ) : (
            <CopyIcon className="size-4" />
          )}
        </Button>
      </Card>
    </>
  );
};

const LazyCodeSheet = ({
  title = "Code",
  queryFn,
  language = "yaml",
  side,
  children = null,
}: LazyCodeSheetProps) => {
  return (
    <Sheet>
      <SheetTrigger asChild>{children}</SheetTrigger>
      <SheetContent
        className="flex flex-col gap-6 sm:max-w-3xl"
        side={side || "right"}
      >
        <CodeSheetContent title={title} queryFn={queryFn} language={language} />
      </SheetContent>
    </Sheet>
  );
};

export default LazyCodeSheet;
