import { CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  MemoryStickIcon as Memory,
  Grid,
  Layers,
  Cable,
  AppWindowIcon,
} from "lucide-react";
import { Separator } from "../ui/separator";

interface NvidiaGpuInfoProps {
  labels: Record<string, string>;
}

export function NvidiaGpuInfoCard({ labels }: NvidiaGpuInfoProps) {
  return (
    <div className="py-4">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="font-mono text-lg font-bold">
          {labels["nvidia.com/gpu.product"]}
        </CardTitle>
      </CardHeader>
      <Separator />
      <CardContent>
        <div className="mt-4 grid grid-cols-2 gap-4">
          <div className="flex flex-col space-y-2">
            <div className="text-primary-foreground/75 flex items-center space-x-2">
              <Memory className="size-4" />
              <span className="text-sm font-medium">显存</span>
            </div>
            <span className="text-lg font-bold">
              {parseInt(labels["nvidia.com/gpu.memory"]) / 1024} GB
            </span>
          </div>
          <div className="flex flex-col space-y-2">
            <div className="text-primary-foreground/75 flex items-center space-x-2">
              <Grid className="size-4" />
              <span className="text-sm font-medium">GPU 数量</span>
            </div>
            <span className="text-lg font-bold">
              {labels["nvidia.com/gpu.count"]}
            </span>
          </div>
          <div className="flex flex-col space-y-2">
            <div className="text-primary-foreground/75 flex items-center space-x-2">
              <Layers className="size-4" />
              <span className="text-sm font-medium">架构</span>
            </div>
            <span className="text-lg font-bold capitalize">
              {labels["nvidia.com/gpu.family"]}
            </span>
          </div>
          <div className="flex flex-col space-y-2">
            <div className="text-primary-foreground/75 flex items-center space-x-2">
              <Cable className="size-4" />
              <span className="text-sm font-medium">驱动版本</span>
            </div>
            <span className="text-lg font-bold">
              {labels["nvidia.com/cuda.driver-version.full"]}
            </span>
          </div>
          {/* CUDA {labels["nvidia.com/cuda.runtime-version.full"]} */}
          <div className="flex flex-col space-y-2">
            <div className="text-primary-foreground/75 flex items-center space-x-2">
              <AppWindowIcon className="size-4" />
              <span className="text-sm font-medium">CUDA</span>
            </div>
            <span className="text-lg font-bold">
              {labels["nvidia.com/cuda.runtime-version.full"]}
            </span>
          </div>
        </div>
      </CardContent>
    </div>
  );
}
