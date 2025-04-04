import { FC, ReactNode } from "react";
import TipBadge from "../badge/TipBadge";
import { cn } from "@/lib/utils";
import { CopyButton } from "../button/copy-button";

interface PageTitleProps {
  title?: string;
  description?: string;
  descriptionCopiable?: boolean;
  children?: ReactNode;
  className?: string;
  tipComponent?: ReactNode;
  tipContent?: ReactNode;
}

const PageTitle: FC<PageTitleProps> = ({
  description,
  descriptionCopiable,
  title,
  children,
  className,
  tipComponent,
  tipContent,
}) => {
  return (
    <div
      className={cn(
        "flex h-12 flex-col justify-between gap-3 md:flex-row md:items-center",
        className,
      )}
    >
      <div>
        <div className="flex items-center gap-1.5 text-xl font-bold">
          <p>{title}</p>
          {tipComponent}
          {tipContent && <TipBadge title={tipContent} />}
        </div>
        {description && (
          <p className="text-muted-foreground flex items-center gap-1 text-sm">
            {description}
            {descriptionCopiable && <CopyButton content={description} />}
          </p>
        )}
      </div>
      {children}
    </div>
  );
};

export default PageTitle;
