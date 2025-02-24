import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import NodeDetail from "@/components/node/NodeDetail";
import { nodeColumns } from "@/components/node/NodeList";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import useNodeQuery from "@/hooks/query/useNodeQuery";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { toast } from "sonner";
import { logger } from "@/utils/loglevel";
import { BanIcon, PaintbrushIcon, TagsIcon } from "lucide-react";
import { useMemo } from "react";
import { useRoutes } from "react-router-dom";
import { apichangeNodeScheduling } from "@/services/api/cluster";
import { useMutation, useQueryClient } from "@tanstack/react-query";
const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索节点名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};
const NodeForAdmin = () => {
  const queryClient = useQueryClient();
  const refetchTaskList = async () => {
    try {
      // 隔 200ms 并行发送所有异步请求
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({ queryKey: ["overview", "nodes"] }),
        ),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };
  const { mutate: handleNodeScheduling } = useMutation({
    mutationFn: apichangeNodeScheduling,
    onSuccess: async () => {
      // 重新获取节点数据
      await refetchTaskList();
      toast.success("操作成功");
    },
    onError: (error) => {
      toast.error(`操作失败: ${error.message}`);
    },
  });
  const nodeQuery = useNodeQuery();

  const columns = useMemo(
    () => [
      ...nodeColumns,
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const nodeId = row.original.name;
          const isReady = row.original.isReady;
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 w-8 p-0">
                  <span className="sr-only">操作</span>
                  <DotsHorizontalIcon className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel className="text-xs text-muted-foreground">
                  操作
                </DropdownMenuLabel>
                <DropdownMenuItem>
                  <TagsIcon className="size-4" />
                  编辑标签
                </DropdownMenuItem>
                <DropdownMenuItem>
                  <PaintbrushIcon className="size-4" />
                  编辑污点
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleNodeScheduling(nodeId)}>
                  <BanIcon className="size-4" />
                  {isReady === "true" ? "禁止调度" : "恢复调度"}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [handleNodeScheduling],
  );

  return (
    <DataTable
      query={nodeQuery}
      columns={columns}
      toolbarConfig={toolbarConfig}
    />
  );
};

export const Component = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <NodeForAdmin />,
    },
    {
      path: ":id",
      element: <NodeDetail />,
    },
  ]);

  return <>{routes}</>;
};
