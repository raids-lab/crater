import { cn } from "@/lib/utils";

export const ProgressBar = ({
  width,
  label,
  className,
}: {
  width: number;
  label?: string;
  className?: string;
}) => {
  const newWidth = width > 100 ? 100 : width;
  return (
    <div
      className={cn(
        "bg-accent text-foreground relative h-2 rounded-md",
        {
          "text-white": width > 90,
        },
        className,
      )}
    >
      <div
        className={cn(
          "h-2 rounded-md transition-all duration-500",
          {
            "bg-highlight-emerald": width <= 20,
            "bg-highlight-sky": width > 20 && width <= 50,
            "bg-highlight-yellow": width > 50 && width <= 70,
            "bg-highlight-orange": width > 70 && width <= 90,
            "bg-highlight-red": width > 90,
          },
          className,
        )}
        style={{ width: `${newWidth}%` }}
      ></div>
      {label && (
        <div className="absolute inset-0 font-mono text-xs font-medium">
          <div className="text-center">{label}</div>
        </div>
      )}
    </div>
  );
};
