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
import { useRecoilValue } from "recoil";
import { REFETCH_INTERVAL } from "@/config/task";

const QuotaCard = ({
  resource,
  ...props
}: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  resource?: ResourceResp;
}) => {
  return (
    <Card className="flex flex-col items-stretch justify-between">
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
        <CardDescription className="flex flex-row items-center justify-start uppercase">
          <div className="mr-1.5 flex h-7 w-7 items-center justify-center rounded-full bg-primary/10">
            <props.icon className="h-4 w-4 text-primary" />
          </div>
          {resource?.label}
        </CardDescription>
        <p className="m-0 font-mono text-xl font-bold">
          {resource?.label === "mem" ? (
            <>
              {(resource?.allocated.amount / 1024 / 1024 / 1024).toFixed(0)}/
              {(resource?.capability.amount / 1024 / 1024 / 1024).toFixed(0)}
              <span className="ml-0.5 text-base">GB</span>
            </>
          ) : (
            <>
              {resource?.allocated.amount}/{resource?.capability.amount}
            </>
          )}
        </p>
      </CardHeader>
      <CardFooter>
        <Progress
          value={
            ((resource?.allocated.amount ?? 0) /
              (resource?.capability.amount ?? 1)) *
            100
          }
          aria-label={resource?.label}
        />
      </CardFooter>
    </Card>
  );
};

const Quota = () => {
  const account = useRecoilValue(globalAccount);
  const { data: quota } = useQuery({
    queryKey: ["context", "quota"],
    queryFn: apiContextQuota,
    select: (res) => res.data.data,
    enabled: account.id !== "",
    refetchInterval: REFETCH_INTERVAL,
  });

  return (
    <div className="grid grid-flow-row grid-cols-2 gap-4 lg:col-span-2">
      <QuotaCard resource={quota?.cpu} icon={CpuIcon} />
      <QuotaCard resource={quota?.memory} icon={MemoryStickIcon} />
      {quota?.gpus?.map((gpu, i) => (
        <QuotaCard key={i} resource={gpu} icon={GpuIcon} />
      ))}
    </div>
  );
};

export default Quota;
