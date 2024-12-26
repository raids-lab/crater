import { FC } from "react";
import TipBadge from "../badge/TipBadge";
import { cn } from "@/lib/utils";

interface PageTitleProps {
  title: string;
  description: string;
  actionArea?: React.ReactNode;
  className?: string;
  isWIP?: boolean;
}

const PageTitle: FC<PageTitleProps> = ({
  description,
  title,
  actionArea,
  className,
  isWIP,
}) => {
  return (
    <div
      className={cn("flex flex-row items-center justify-between", className)}
    >
      <div>
        <h1 className="flex items-center text-xl font-bold leading-relaxed">
          {title}
          {isWIP && <TipBadge className="ml-1.5" />}
        </h1>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
      {actionArea}
    </div>
  );
};

export default PageTitle;
