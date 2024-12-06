import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { MemoryStickIcon as Memory, Grid, Layers, Cable } from "lucide-react";

interface NvidiaGpuInfoProps {
  labels: Record<string, string>;
}

export function NvidiaGpuInfoCard({ labels }: NvidiaGpuInfoProps) {
  return (
    <Card className="w-full max-w-md">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-lg font-bold text-primary">
          {labels["nvidia.com/gpu.product"]}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="mt-4 grid grid-cols-2 gap-4">
          <div className="flex flex-col space-y-2">
            <div className="flex items-center space-x-2 text-muted-foreground">
              <Memory className="h-4 w-4" />
              <span className="text-sm font-medium">显存</span>
            </div>
            <span className="text-lg font-bold">
              {parseInt(labels["nvidia.com/gpu.memory"]) / 1024} GB
            </span>
          </div>
          <div className="flex flex-col space-y-2">
            <div className="flex items-center space-x-2 text-muted-foreground">
              <Grid className="h-4 w-4" />
              <span className="text-sm font-medium">GPU 数量</span>
            </div>
            <span className="text-lg font-bold">
              {labels["nvidia.com/gpu.count"]}
            </span>
          </div>
          <div className="flex flex-col space-y-2">
            <div className="flex items-center space-x-2 text-muted-foreground">
              <Layers className="h-4 w-4" />
              <span className="text-sm font-medium">架构</span>
            </div>
            <span className="text-lg font-bold capitalize">
              {labels["nvidia.com/gpu.family"]}
            </span>
          </div>
          <div className="flex flex-col space-y-2">
            <div className="flex items-center space-x-2 text-muted-foreground">
              <Cable className="h-4 w-4" />
              <span className="text-sm font-medium">驱动版本</span>
            </div>
            <span className="text-lg font-bold">
              {labels["nvidia.com/cuda.driver-version.full"]}
            </span>
          </div>
        </div>
        <div className="mt-4 flex items-center space-x-2">
          <Badge variant="secondary">
            CUDA {labels["nvidia.com/cuda.runtime-version.full"]}
          </Badge>
        </div>
      </CardContent>
    </Card>
  );
}
