import { useQuery } from "@tanstack/react-query";
import { apiGetNodes } from "@/services/api/cluster";
import { convertKResourceToResource } from "@/utils/resource";
import { ClusterNodeInfo } from "@/components/node/NodeList";

const useNodeQuery = (onlyWorker?: boolean) => {
  return useQuery({
    queryKey: ["overview", "nodes"],
    queryFn: apiGetNodes,
    select: (res) =>
      res.data.data.rows
        .sort((a, b) => a.name.localeCompare(b.name))
        .sort((a, b) => {
          // 按照 type 排序，优先 hygon => shenwei => yitian => 空字符串
          const typeOrder = ["hygon", "shenwei", "yitian", ""];
          return typeOrder.indexOf(a.type) - typeOrder.indexOf(b.type);
        })
        .filter((x) => !onlyWorker || x.role === "worker")
        .map((x) => {
          const cpuCapacity = convertKResourceToResource(
            "cpu",
            x.capacity && x.capacity.cpu,
          );
          const cpuAllocated = convertKResourceToResource(
            "cpu",
            x.allocated && x.allocated.cpu,
          );
          const memCapacity = convertKResourceToResource(
            "memory",
            x.capacity && x.capacity.memory,
          );
          const memAllocated = convertKResourceToResource(
            "memory",
            x.allocated && x.allocated.memory,
          );
          const gpuCapacity = convertKResourceToResource(
            "nvidia.com/gpu",
            x.capacity && x.capacity["nvidia.com/gpu"],
          );
          const gpuAllocated = convertKResourceToResource(
            "nvidia.com/gpu",
            x.allocated && x.allocated["nvidia.com/gpu"],
          );

          const info: ClusterNodeInfo = {
            type: x.type,
            name: x.name,
            isReady: x.isReady,
            taint: x.taint,
            role: x.role,
            labels: x.labels,
            podCount: x.podCount,
          };

          if (cpuCapacity !== undefined && cpuAllocated !== undefined) {
            info.cpu = {
              percent: (cpuAllocated / cpuCapacity) * 100,
              description: `${cpuAllocated.toFixed(1)}/${cpuCapacity.toFixed(0)}`,
            };
          }
          if (memCapacity !== undefined && memAllocated !== undefined) {
            info.memory = {
              percent: (memAllocated / memCapacity) * 100,
              description: `${memAllocated.toFixed(1)}/${memCapacity.toFixed(0)}`,
            };
          }
          if (gpuCapacity !== undefined && gpuAllocated !== undefined) {
            info.gpu = {
              percent: (gpuAllocated / gpuCapacity) * 100,
              description: `${gpuAllocated}/${gpuCapacity}`,
            };
          }

          return info;
        }),
  });
};

export default useNodeQuery;
