// i18n-processed-v1.1.0 (no translatable strings)
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { LucideIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import CardTitle from "../label/CardTitle";

type AccordionCardProps = React.HTMLAttributes<HTMLDivElement> & {
  cardTitle: string;
  icon: LucideIcon;
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
          <AccordionTrigger className="mx-6 cursor-pointer leading-none tracking-tight hover:no-underline">
            <CardTitle icon={props.icon}>{cardTitle}</CardTitle>
          </AccordionTrigger>
          <AccordionContent className="px-6">{children}</AccordionContent>
        </AccordionItem>
      </Accordion>
    </Card>
  );
};

export default AccordionCard;
