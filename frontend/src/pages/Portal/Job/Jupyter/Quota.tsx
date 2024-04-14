import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { apiContextQuota } from "@/services/api/context";
import { useQuery } from "@tanstack/react-query";
import { CpuIcon, CylinderIcon, MemoryStickIcon } from "lucide-react";
import { useMemo } from "react";

const Quota = () => {
  const { data: quota, isLoading } = useQuery({
    queryKey: ["aitask", "quota"],
    queryFn: apiContextQuota,
    select: (res) => res.data.data,
  });

  const quotaData = useMemo(() => {
    if (quota === undefined || isLoading) return [];
    return [
      {
        icon: CpuIcon,
        resource: "CPU",
        progress: {
          width: (quota.cpu / quota.cpuReq) * 100,
          label: `${quota.cpu}/${quota.cpuReq}`,
        },
        usage: `${quota.cpu}/${quota.cpuReq}`,
        unit: "",
        soft: `${quota.cpuReq}`,
      },
      {
        icon: CpuIcon,
        resource: "GPU",
        progress: {
          width: (quota.gpu / quota.gpuReq) * 100,
          label: `${quota.gpu}/${quota.gpuReq}`,
        },
        usage: `${quota.gpu}/${quota.gpuReq}`,
        unit: "",
        soft: `${quota.gpuReq}`,
      },
      {
        icon: MemoryStickIcon,
        resource: "内存",
        progress: {
          width: (quota.mem / quota.memReq) * 100,
          label: `${quota.mem}/${quota.memReq}`,
        },
        usage: `${quota.mem}/${quota.memReq}`,
        unit: "GB",
        soft: `${quota.memReq}`,
      },
      {
        icon: CylinderIcon,
        resource: "存储",
        progress: {
          width: (quota.storage / quota.storage) * 100,
          label: `${quota.storage}/${quota.storage}`,
        },
        usage: `${quota.storage}/${quota.storage}`,
        unit: "GB",
        soft: `${quota.storage}`,
      },
    ];
  }, [quota, isLoading]);

  return (
    <div className="grid grid-flow-row grid-cols-2 gap-4 md:col-span-2">
      {quotaData.map((q, i) => (
        <Card key={i} className="flex flex-col items-stretch justify-between">
          <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
            <CardDescription className="flex flex-row items-center justify-start">
              <q.icon className="mr-1.5 h-4 w-4" />
              <p>{q.resource}</p>
            </CardDescription>
            <p className="m-0 font-mono text-2xl font-bold leading-none">
              {q.usage}
              <span className="ml-0.5 text-base">{q.unit}</span>
            </p>
          </CardHeader>
          <CardFooter>
            <Progress value={q.progress.width} aria-label={q.progress.label} />
          </CardFooter>
        </Card>
      ))}
    </div>
  );
};

export default Quota;
