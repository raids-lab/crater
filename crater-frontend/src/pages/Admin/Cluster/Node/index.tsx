import { useState } from "react";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import NodeDetail from "@/components/node/NodeDetail";
import { nodeColumns } from "@/components/node/NodeList";
import { Button } from "@/components/ui/button";
import AccountSelect from "./AccountList";
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
import { BanIcon, ZapIcon, Users } from "lucide-react";
import { useMemo } from "react";
import { useRoutes } from "react-router-dom";
import {
  apichangeNodeScheduling,
  apiAddNodeTaint,
  apiDeleteNodeTaint,
} from "@/services/api/cluster";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";

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
  const [open, setOpen] = useState(false);
  const [selectedAccount, setSelectedAccount] = useState("");
  const [selectedNode, setSelectedNode] = useState("");
  const [isOccupation, setIsOccupation] = useState(true);
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
  const { mutate: addNodeTaint } = useMutation({
    mutationFn: apiAddNodeTaint,
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("节点占有成功");
    },
    onError: (error) => {
      toast.error(`节点占有失败: ${error.message}`);
    },
  });
  const { mutate: deleteNodeTaint } = useMutation({
    mutationFn: apiDeleteNodeTaint,
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("取消节点占有成功");
    },
    onError: (error) => {
      toast.error(`取消节点占有失败: ${error.message}`);
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
                <DropdownMenuLabel className="text-muted-foreground text-xs">
                  操作
                </DropdownMenuLabel>
                <DropdownMenuItem
                  onClick={() => {
                    setSelectedNode(nodeId);
                    setIsOccupation(true);
                    setOpen(true);
                  }}
                >
                  <Users size={16} strokeWidth={2} />
                  账户占有
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => {
                    setSelectedNode(nodeId);
                    setIsOccupation(false);
                    setOpen(true);
                  }}
                >
                  <Users size={16} strokeWidth={2} />
                  取消独占
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleNodeScheduling(nodeId)}>
                  {isReady === "true" ? (
                    <BanIcon className="size-4" />
                  ) : (
                    <ZapIcon className="size-4" />
                  )}
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

  const handleOccupation = () => {
    const taintcontent = `crater.raids.io/account=${selectedAccount}:NoSchedule`;
    const taint = {
      name: selectedNode,
      taint: taintcontent,
    };
    if (isOccupation) {
      addNodeTaint(taint);
    } else {
      deleteNodeTaint(taint);
    }
    setOpen(false);
  };

  return (
    <>
      <DataTable
        info={{
          title: "节点管理",
          description: "在这里修改节点的调度策略、是否由某些账户独享等",
        }}
        query={nodeQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
      />
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{isOccupation ? "账户占有" : "取消占有"}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <AccountSelect
              value={selectedAccount}
              onChange={setSelectedAccount}
            />
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">取消</Button>
            </DialogClose>
            <Button variant="default" onClick={handleOccupation}>
              提交
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
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
