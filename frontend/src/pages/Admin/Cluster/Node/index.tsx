import { DataTable } from "@/components/custom/DataTable";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import NodeDetail from "@/components/custom/Node/NodeDetail";
import { nodeColumns } from "@/components/custom/Node/NodeList";
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
import { BanIcon, PaintbrushIcon, TagsIcon } from "lucide-react";
import { useMemo } from "react";
import { useRoutes } from "react-router-dom";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索节点名称",
    key: "name",
  },
  filterOptions: [],
  getHeader: (x) => x,
};

const NodeForAdmin = () => {
  const nodeQuery = useNodeQuery();

  const columns = useMemo(
    () => [
      ...nodeColumns,
      {
        id: "actions",
        enableHiding: false,
        cell: () => {
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 w-8 p-0">
                  <span className="sr-only">操作</span>
                  <DotsHorizontalIcon className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>操作</DropdownMenuLabel>
                <DropdownMenuItem>
                  <TagsIcon className="h-4 w-4" />
                  编辑标签
                </DropdownMenuItem>
                <DropdownMenuItem>
                  <PaintbrushIcon className="h-4 w-4" />
                  编辑污点
                </DropdownMenuItem>
                <DropdownMenuItem>
                  <BanIcon className="h-4 w-4" />
                  禁止调度
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [],
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
