import { FC, ReactNode } from "react";
import TipBadge from "../badge/TipBadge";
import { cn } from "@/lib/utils";

interface PageTitleProps {
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  className?: string;
  isWIP?: boolean;
  tipContent?: ReactNode;
}

const PageTitle: FC<PageTitleProps> = ({
  description,
  title,
  children,
  className,
  isWIP,
  tipContent,
}) => {
  return (
    <div
      className={cn(
        "flex flex-col justify-between gap-3 md:flex-row md:items-center",
        className,
      )}
    >
      <div>
        <h1 className="flex items-center text-xl leading-relaxed font-bold">
          {title}
          {isWIP && <TipBadge className="ml-1.5" title={tipContent} />}
        </h1>
        {description && (
          <p className="text-muted-foreground text-sm">{description}</p>
        )}
      </div>
      {children}
    </div>
  );
};

export default PageTitle;
