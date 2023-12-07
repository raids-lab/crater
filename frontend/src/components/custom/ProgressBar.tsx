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
  return (
    <div className="relative h-4 rounded bg-dashboard text-foreground dark:bg-sidebar-item">
      <div
        className={cn("h-4 rounded", {
          "bg-green-400 dark:bg-green-700": width <= 30,
          "bg-yellow-400 dark:bg-yellow-700": width > 30 && width <= 60,
          "bg-orange-400 dark:bg-orange-700": width > 60 && width <= 80,
          "bg-red-500 dark:bg-rose-700": width > 80,
        })}
        style={{ width: `${width}%` }}
      ></div>
      {first && second ? (
        <div className="absolute inset-0 grid grid-cols-7 gap-1 font-mono text-xs font-medium ">
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
