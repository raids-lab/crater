import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import { ReactNode } from "react";

export default function WarningAlert({
  title,
  description,
  className,
}: {
  title: ReactNode;
  description: ReactNode;
  className?: string;
}) {
  return (
    <Alert className={cn("border-orange-600 bg-orange-600/15", className)}>
      <AlertTitle className="text-highlight-orange">{title}</AlertTitle>
      <AlertDescription>{description}</AlertDescription>
    </Alert>
  );
}
