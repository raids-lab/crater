import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { DefaultQuota, apiContextQuota } from "@/services/api/context";
import { getAiResource } from "@/utils/resource";
import { globalAccount } from "@/utils/store";
import { useQuery } from "@tanstack/react-query";
import { CpuIcon, HardDriveIcon, MemoryStickIcon } from "lucide-react";
import { useMemo } from "react";
import { useRecoilValue } from "recoil";

const Quota = () => {
  const account = useRecoilValue(globalAccount);
  const { data: quota } = useQuery({
    queryKey: ["context", "quota"],
    queryFn: apiContextQuota,
    select: (res) => res.data.data || DefaultQuota,
    enabled: account.id !== "",
  });

  const quotaData = useMemo(() => {
    const allocated = getAiResource(quota ? quota.allocated : undefined);
    const capability = getAiResource(quota ? quota.capability : undefined);
    return [
      {
        icon: CpuIcon,
        resource: "CPU",
        progress: {
          width: ((allocated.cpu ?? 0) / (capability.cpu ?? 1)) * 100,
          label: `${allocated.cpu}/${capability.cpu}`,
        },
        usage: `${allocated.cpu}/${capability.cpu}`,
        unit: "",
        soft: `${capability.cpu}`,
      },
      {
        icon: CpuIcon,
        resource: "GPU",
        progress: {
          width: ((allocated.gpu ?? 0) / (capability.gpu ?? 1)) * 100,
          label: `${allocated.gpu}/${capability.gpu}`,
        },
        usage: `${allocated.gpu}/${capability.gpu}`,
        unit: "",
        soft: `${capability.gpu}`,
      },
      {
        icon: MemoryStickIcon,
        resource: "内存",
        progress: {
          width:
            ((allocated.memoryNum ?? 0) / (capability.memoryNum ?? 1)) * 100,
          label: `${allocated.memoryNum}/${capability.memoryNum}`,
        },
        usage: `${allocated.memoryNum}/${capability.memoryNum}`,
        unit: "GB",
        soft: `${capability.memoryNum}`,
      },
      {
        icon: HardDriveIcon,
        resource: "个人存储",
        progress: {
          width: (10 / 50) * 100,
          label: `${10}/${50}`,
        },
        usage: `10/50`,
        unit: "GB",
        soft: `${50}`,
      },
    ];
  }, [quota]);

  return (
    <div className="grid grid-flow-row grid-cols-2 gap-4 md:col-span-2">
      {quotaData.map((q, i) => (
        <Card key={i} className="flex flex-col items-stretch justify-between">
          <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
            <CardDescription className="flex flex-row items-center justify-start">
              <q.icon className="mr-1.5 h-4 w-4" />
              {q.resource}
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
