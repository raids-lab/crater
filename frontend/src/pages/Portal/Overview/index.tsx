import { useMemo, type FC } from "react";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useQuery } from "@tanstack/react-query";
import { apiAiTaskQuota } from "@/services/api/aiTask";
import { getAiResource } from "@/utils/resource";
import { ProgressBar } from "@/components/custom/ProgressBar";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import Tasks from "./Tasks";
import Status from "./Status";
import "./index.css";
import { logger } from "@/utils/loglevel";

interface Quota {
  resource: string;
  progress: {
    width: number;
    label: string;
  };
  usage: string;
  soft: string;
}

export const Component: FC = () => {
  const { data: quota, isLoading } = useQuery({
    queryKey: ["aitask", "quota"],
    queryFn: apiAiTaskQuota,
    select: (res) => ({
      hard: getAiResource(res.data.data.hard),
      hardUsed: getAiResource(res.data.data.hardUsed),
      softUsed: getAiResource(res.data.data.softUsed),
    }),
  });

  const columns: ColumnDef<Quota>[] = [
    {
      accessorKey: "resource",
      header: () => <></>,
      cell: ({ row }) => (
        <div className="ml-2 font-mono">{row.getValue("resource")}</div>
      ),
    },
    {
      accessorKey: "progress",
      header: () => <div className="text-xs">固定配额用量</div>,
      cell: ({ row }) => {
        const progress: Quota["progress"] = row.getValue("progress");
        return (
          <div className="w-40">
            <ProgressBar width={progress.width * 100} label={progress.label} />
          </div>
        );
      },
    },
    // {
    //   accessorKey: "usage",
    //   header: () => <div className="text-xs">Used/Hard</div>,
    //   cell: ({ row }) => (
    //     <div className="font-mono">{row.getValue("usage")}</div>
    //   ),
    // },
    {
      accessorKey: "soft",
      header: () => <div className="text-xs">弹性配额用量</div>,
      cell: ({ row }) => (
        <div className="font-mono">{row.getValue("soft")}</div>
      ),
    },
  ];

  const data = useMemo<Quota[]>(() => {
    if (isLoading || !quota) {
      return [];
    }
    const tableData = [
      {
        resource: "CPU",
        progress: {
          width:
            quota.hardUsed.cpu && quota.hard.cpu
              ? quota.hardUsed.cpu / quota.hard.cpu
              : 0,
          label: `${quota.hardUsed.cpu}/${quota.hard.cpu}`,
        },
        usage: `${quota.hardUsed.cpu}/${quota.hard.cpu}`,
        soft: `${quota.softUsed.cpu}`,
      },
      {
        resource: "GPU",
        progress: {
          width:
            quota.hardUsed.gpu && quota.hard.gpu
              ? quota.hardUsed.gpu / quota.hard.gpu
              : 0,
          label: `${quota.hardUsed.gpu}/${quota.hard.gpu}`,
        },
        usage: `${quota.hardUsed.gpu}/${quota.hard.gpu}`,
        soft: `${quota.softUsed.gpu}`,
      },
      {
        resource: "Memory",
        progress: {
          width:
            quota.hardUsed.memoryNum && quota.hard.memoryNum
              ? quota.hardUsed.memoryNum / quota.hard.memoryNum
              : 0,
          label: `${quota.hardUsed.memoryNum}/${quota.hard.memory}`,
        },
        usage: `${quota.hardUsed.memoryNum}/${quota.hard.memory}`,
        soft: `${quota.softUsed.memory}`,
      },
    ];
    logger.debug("Table Data: ", tableData);
    return tableData;
  }, [quota, isLoading]);

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <div className="grid grid-flow-row-dense gap-6 md:grid-cols-2">
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>作业队列</CardTitle>
          {/* <CardDescription>
            <p>1. 所有正在系统中运行的作业</p>
            <p>2. (如果有) 用户自己正在运行的作业</p>
          </CardDescription> */}
        </CardHeader>
        <CardContent>
          <Status />
        </CardContent>
      </Card>
      <Card className="col-span-1">
        <CardHeader>
          <CardTitle>我的配额</CardTitle>
          <CardDescription>
            <span>1. 查看用户个人账户的资源配额</span>
            <br />
            <span>2. 查看用户在不同账户下的资源配额</span>
          </CardDescription>
        </CardHeader>
        <CardContent className="text-base">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="hover:bg-background">
                  {headerGroup.headers.map((header) => {
                    return (
                      <TableHead key={header.id} colSpan={header.colSpan}>
                        {header.isPlaceholder
                          ? null
                          : flexRender(
                              header.column.columnDef.header,
                              header.getContext(),
                            )}
                      </TableHead>
                    );
                  })}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id} className="hover:bg-background">
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="h-24 text-center"
                  >
                    暂无数据
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <Card className="col-span-full">
        <CardHeader>
          <CardTitle>资源使用情况 [WIP]</CardTitle>
          <CardDescription>
            <span>1. 当前整体集群的资源使用情况</span>
            <br />
            <span>
              2. 查看整体集群的历史资源使用情况 (详细信息对接prometheus)
            </span>
            <br />
            <span>3. 查看本用户的历史资源使用情况 (汇总统计)</span>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tasks />
        </CardContent>
      </Card>
    </div>
  );
};
