import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Card } from "@/components/ui/card";

type AccordionCardProps = React.HTMLAttributes<HTMLDivElement> & {
  cardTitle: string;
  value?: string;
  setValue?: (value: string) => void;
};

const AccordionCard = ({
  cardTitle,
  value,
  setValue,
  children,
  ...props
}: AccordionCardProps) => {
  return (
    <Card {...props}>
      <Accordion
        type="single"
        collapsible
        value={value}
        onValueChange={setValue}
      >
        <AccordionItem value={cardTitle} className="border-b-0 py-2">
          <AccordionTrigger className="before:bg-primary/75 relative mx-6 pl-2.5 text-base leading-none font-semibold tracking-tight before:absolute before:left-0 before:h-4 before:w-1 before:rounded-full hover:no-underline">
            {cardTitle}
          </AccordionTrigger>
          <AccordionContent className="px-6">{children}</AccordionContent>
        </AccordionItem>
      </Accordion>
    </Card>
  );
};

export default AccordionCard;
