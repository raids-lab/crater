import { useState, useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGetPodContainers, ContainerInfo } from "@/services/api/tool";
import LoadingCircleIcon from "../icon/LoadingCircleIcon";
import {
  ContainerSelect,
  PodNamespacedName,
  TableCellForm,
} from "./PodContainerDialog";
import { LogCard } from "./LogDialog";

export default function DetailPageLog({
  namespacedName,
}: {
  namespacedName?: PodNamespacedName;
}) {
  const { namespace, name: podName } = namespacedName ?? {};
  const queryClient = useQueryClient();

  const [selectedContainer, setSelectedContainer] = useState<
    ContainerInfo | undefined
  >();

  const { data: containers } = useQuery({
    queryKey: ["log", "containers", namespace, podName],
    queryFn: () => apiGetPodContainers(namespace, podName),
    select: (res) => res.data.data.containers.filter((c) => c.name !== ""),
    enabled: !!namespace && !!podName,
  });

  useEffect(() => {
    for (const container of containers || []) {
      if (!container.isInitContainer) {
        setSelectedContainer((prev) => {
          return prev || container;
        });
        void queryClient.invalidateQueries({ queryKey: ["logtext"] });
        break;
      }
    }
  }, [containers, queryClient]);

  return (
    <>
      {podName === "" ? (
        <div className="flex h-[calc(100vh_-304px)] w-full items-center justify-center">
          镜像构建Pod已删除，无法查看日志
        </div>
      ) : !containers || !selectedContainer ? (
        <div className="flex h-[calc(100vh_-304px)] w-full items-center justify-center">
          <LoadingCircleIcon />
        </div>
      ) : (
        <div className="grid h-[calc(100vh_-304px)] w-full gap-6 md:grid-cols-3 xl:grid-cols-4">
          {namespacedName && selectedContainer && (
            <LogCard
              namespacedName={namespacedName}
              selectedContainer={selectedContainer}
            />
          )}
          <div className="space-y-4">
            <ContainerSelect
              currentContainer={selectedContainer}
              setCurrentContainer={setSelectedContainer}
              containers={containers}
            />
            <fieldset className="border-input hidden h-[calc(100vh_-378px)] max-h-full gap-6 overflow-y-auto rounded-lg border p-4 shadow-xs md:grid">
              <legend className="-ml-1 px-2 text-sm font-medium">
                {selectedContainer.isInitContainer ? "初始化容器" : "容器信息"}
              </legend>
              <TableCellForm selectedContainer={selectedContainer} />
            </fieldset>
          </div>
        </div>
      )}
    </>
  );
}
