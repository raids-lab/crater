import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { ResourceResp, apiContextQuota } from "@/services/api/context";
import { globalAccount } from "@/utils/store";
import { useQuery } from "@tanstack/react-query";
import { CpuIcon, MemoryStickIcon } from "lucide-react";
import GpuIcon from "@/components/icon/GpuIcon";
import { useAtomValue } from "jotai";
import { REFETCH_INTERVAL } from "@/config/task";
import { useMemo } from "react";
import { cn } from "@/lib/utils";

const showAmount = (allocated: number, label?: string) => {
  if (label === "mem") {
    return (allocated / 1024 / 1024 / 1024).toFixed(0);
  }
  return allocated;
};

const showUnit = (label?: string) => {
  if (label === "mem") {
    return "GB";
  } else if (label === "cpu") {
    return "核";
  }
  return "卡";
};

const QuotaCard = ({
  resource,
  ...props
}: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  resource?: ResourceResp;
}) => {
  const allocated = resource?.allocated?.amount ?? 0;
  const quota = resource?.deserved?.amount ?? resource?.capability?.amount ?? 1;
  const [progress, overflow] = useMemo(() => {
    const progress = (allocated / quota) * 100;
    const overflow = progress > 100;
    return [overflow ? 100 : progress, overflow];
  }, [allocated, quota]);
  return (
    <Card className="flex flex-col items-stretch justify-between">
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
        <CardDescription className="flex flex-row items-center justify-start uppercase">
          <div className="mr-1.5 flex h-7 w-7 items-center justify-center rounded-full bg-primary/10">
            <props.icon className="h-4 w-4 text-primary" />
          </div>
          {resource?.label}
        </CardDescription>
        <div className="font-sans text-xs text-muted-foreground">
          已用
          <span
            className={cn("mx-0.5 font-mono text-xl font-bold text-primary", {
              "text-orange-500": overflow,
            })}
          >
            {showAmount(allocated, resource?.label)}
          </span>
          {showUnit(resource?.label)}
        </div>
      </CardHeader>
      <CardFooter className="flex flex-col gap-2">
        <Progress
          value={progress}
          aria-label={resource?.label}
          className={cn({ "bg-orange-500/20 [&>*]:bg-orange-400": overflow })}
        />
        {!!resource?.capability &&
          !!resource?.guarantee &&
          !!resource?.deserved && (
            <div className="flex w-full flex-row-reverse items-center justify-between text-xs">
              {resource?.capability?.amount !== undefined && (
                <p className="text-orange-500">
                  上限:{" "}
                  {showAmount(resource?.capability?.amount, resource?.label)}
                </p>
              )}
              {resource?.deserved?.amount !== undefined && (
                <p className="text-teal-500">
                  应得:{" "}
                  {showAmount(resource?.deserved?.amount, resource?.label)}
                </p>
              )}
              {resource?.allocated?.amount !== undefined && (
                <p className="text-sky-500">
                  已用:{" "}
                  {showAmount(resource?.allocated?.amount, resource?.label)}
                </p>
              )}
              {resource?.guarantee?.amount !== undefined &&
                resource?.guarantee?.amount > 0 && (
                  <p className="text-slate-500">
                    保证:{" "}
                    {showAmount(resource?.guarantee?.amount, resource?.label)}
                  </p>
                )}
            </div>
          )}
      </CardFooter>
    </Card>
  );
};

const Quota = () => {
  const account = useAtomValue(globalAccount);
  const { data: quota } = useQuery({
    queryKey: ["context", "quota"],
    queryFn: apiContextQuota,
    select: (res) => res.data.data,
    enabled: account.queue !== "",
    refetchInterval: REFETCH_INTERVAL,
  });

  return (
    <>
      <QuotaCard resource={quota?.cpu} icon={CpuIcon} />
      <QuotaCard resource={quota?.memory} icon={MemoryStickIcon} />
      {quota?.gpus?.map((gpu, i) => (
        <QuotaCard key={i} resource={gpu} icon={GpuIcon} />
      ))}
    </>
  );
};

export default Quota;
