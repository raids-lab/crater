import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { useCallback, useMemo } from "react";

type AccordionCardProps = React.HTMLAttributes<HTMLDivElement> & {
  cardTitle: string;
  open: boolean;
  setOpen?: (open: boolean) => void;
};

const AccordionCard = ({
  cardTitle,
  open,
  setOpen,
  children,
  className,
  ...props
}: AccordionCardProps) => {
  // 处理值变化，转换回布尔值
  const handleValueChange = useCallback(
    (newValue: string) => {
      if (setOpen) {
        setOpen(newValue === cardTitle);
      }
    },
    [setOpen, cardTitle],
  );

  // 将 open 布尔值转换为 Accordion 需要的 value 字符串
  const value = useMemo(() => (open ? cardTitle : ""), [open, cardTitle]);

  return (
    <Card className={cn(className, "p-0")} {...props}>
      <Accordion
        type="single"
        collapsible
        value={value}
        onValueChange={handleValueChange}
      >
        <AccordionItem value={cardTitle} className="border-b-0 py-2">
          <AccordionTrigger className="before:bg-primary/75 relative mx-6 cursor-pointer pl-2.5 text-base leading-none font-semibold tracking-tight before:absolute before:left-0 before:h-4 before:w-1 before:rounded-full hover:no-underline">
            {cardTitle}
          </AccordionTrigger>
          <AccordionContent className="px-6">{children}</AccordionContent>
        </AccordionItem>
      </Accordion>
    </Card>
  );
};

export default AccordionCard;
