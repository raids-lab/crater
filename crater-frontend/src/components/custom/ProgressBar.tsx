import { cn } from "@/lib/utils";

export const ProgressBar = ({
  width,
  label,
}: {
  width: number;
  label: string;
}) => {
  // dramActiveAvg: 19.41%, split by :
  const [first, second] = label.split(":");
  const newWidth = width > 100 ? 100 : width;
  return (
    <div
      className={cn("bg-accent/50 text-foreground relative h-4 rounded", {
        "text-white": width > 90,
      })}
    >
      <div
        className={cn("h-4 rounded transition-all duration-500", {
          "bg-green-400 dark:bg-green-700": width <= 30,
          "bg-yellow-400 dark:bg-yellow-700": width > 30 && width <= 60,
          "bg-orange-400 dark:bg-orange-700": width > 60 && width <= 90,
          "bg-red-500 dark:bg-rose-700": width > 90,
        })}
        style={{ width: `${newWidth}%` }}
      ></div>
      {first && second ? (
        <div className="absolute inset-0 grid grid-cols-7 gap-1 font-mono text-xs font-medium">
          <div className="col-span-4 text-right">{first}:</div>
          <div className="col-span-3">{second}</div>
        </div>
      ) : (
        <div className="absolute inset-0 font-mono text-xs font-medium">
          <div className="text-center">{first}</div>
        </div>
      )}
    </div>
  );
};
