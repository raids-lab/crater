import { useMemo, type FC, useState, useEffect } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
} from "@/components/ui/pagination";
import {
  ChevronLeft,
  ChevronRight,
  File,
  ListFilter,
  MoreVertical,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { useQuery } from "@tanstack/react-query";
import { apiGetNodes } from "@/services/api/node";
import {
  ColumnDef,
  PaginationState,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
  Table as TableType,
} from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { UsageCell } from "@/pages/Admin/Cluster/Node";
import { getAiResource } from "@/utils/resource";
import { statuses, getHeader, VolcanoJobInfo } from "@/pages/Admin/Job/Volcano";
import { TableDate } from "@/components/custom/TableDate";
import { cn } from "@/lib/utils";
import { apiAdminTaskListByType } from "@/services/api/admin/task";
interface ResourceInfo {
  percent: number;
  description: string;
}

interface ClusterNodeInfo {
  name: string;
  cpu: ResourceInfo;
  memory: ResourceInfo;
  gpu: ResourceInfo;
}
type info = ClusterNodeInfo | VolcanoJobInfo;
const NodeHome = <T extends info>({
  columns,
  table,
  isLoading,
}: {
  columns: ColumnDef<T>[];
  table: TableType<T>;
  isLoading: boolean;
}) => {
  return (
    <Table>
      <TableHeader>
        {table.getHeaderGroups().map((headerGroup) => (
          <TableRow key={headerGroup.id} className="bg-accent hover:bg-accent">
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
            <TableRow
              key={row.id}
              data-state={row.getIsSelected() && "selected"}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
          ))
        ) : (
          <>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-60">
                  <svg
                    aria-hidden="true"
                    className="m-auto h-8 w-8 animate-spin fill-primary text-gray-200 dark:text-gray-600"
                    viewBox="0 0 100 101"
                    fill="none"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      d="M100 50.5908C100 78.2051 77.6142 100.591 50 100.591C22.3858 100.591 0 78.2051 0 50.5908C0 22.9766 22.3858 0.59082 50 0.59082C77.6142 0.59082 100 22.9766 100 50.5908ZM9.08144 50.5908C9.08144 73.1895 27.4013 91.5094 50 91.5094C72.5987 91.5094 90.9186 73.1895 90.9186 50.5908C90.9186 27.9921 72.5987 9.67226 50 9.67226C27.4013 9.67226 9.08144 27.9921 9.08144 50.5908Z"
                      fill="currentColor"
                    />
                    <path
                      d="M93.9676 39.0409C96.393 38.4038 97.8624 35.9116 97.0079 33.5539C95.2932 28.8227 92.871 24.3692 89.8167 20.348C85.8452 15.1192 80.8826 10.7238 75.2124 7.41289C69.5422 4.10194 63.2754 1.94025 56.7698 1.05124C51.7666 0.367541 46.6976 0.446843 41.7345 1.27873C39.2613 1.69328 37.813 4.19778 38.4501 6.62326C39.0873 9.04874 41.5694 10.4717 44.0505 10.1071C47.8511 9.54855 51.7191 9.52689 55.5402 10.0491C60.8642 10.7766 65.9928 12.5457 70.6331 15.2552C75.2735 17.9648 79.3347 21.5619 82.5849 25.841C84.9175 28.9121 86.7997 32.2913 88.1811 35.8758C89.083 38.2158 91.5421 39.6781 93.9676 39.0409Z"
                      fill="currentFill"
                    />
                  </svg>
                </TableCell>
              </TableRow>
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-40 text-center"
                >
                  暂无数据
                </TableCell>
              </TableRow>
            )}
          </>
        )}
      </TableBody>
    </Table>
  );
};

const columns: ColumnDef<ClusterNodeInfo>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"节点名称"} />
    ),
    cell: ({ row }) => <div>{row.getValue("name")}</div>,
    enableSorting: false,
  },
  {
    accessorKey: "cpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"CPU"} />
    ),
    cell: ({ row }) => <UsageCell value={row.getValue<ResourceInfo>("cpu")} />,
    enableSorting: false,
  },
  {
    accessorKey: "memory",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"Memory"} />
    ),
    cell: ({ row }) => (
      <UsageCell value={row.getValue<ResourceInfo>("memory")} />
    ),
    enableSorting: false,
  },
  {
    accessorKey: "gpu",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={"GPU"} />
    ),
    cell: ({ row }) => <UsageCell value={row.getValue<ResourceInfo>("gpu")} />,
    enableSorting: false,
  },
];

const jobColumns: ColumnDef<VolcanoJobInfo>[] = [
  {
    accessorKey: "jobName",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("jobName")} />
    ),
    cell: ({ row }) => <div>{row.getValue("jobName")}</div>,
  },
  {
    accessorKey: "jobType",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("jobType")} />
    ),
    cell: ({ row }) => <div>{row.getValue("jobType")}</div>,
  },
  {
    accessorKey: "userName",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("userName")} />
    ),
    cell: ({ row }) => <div>{row.getValue("userName")}</div>,
  },
  {
    accessorKey: "queue",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("queue")} />
    ),
    cell: ({ row }) => <div>{row.getValue("queue")}</div>,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("status")} />
    ),
    cell: ({ row }) => {
      const status = statuses.find(
        (status) => status.value === row.getValue("status"),
      );
      if (!status) {
        return null;
      }
      return (
        <div className="flex flex-row items-center justify-start">
          <div
            className={cn("flex h-3 w-3 rounded-full", {
              "bg-purple-500 hover:bg-purple-400": status.value === "Queueing",
              "bg-slate-500 hover:bg-slate-400": status.value === "Created",
              "bg-pink-500 hover:bg-pink-400": status.value === "Pending",
              "bg-sky-500 hover:bg-sky-400": status.value === "Running",
              "bg-red-500 hover:bg-red-400": status.value === "Failed",
              "bg-emerald-500 hover:bg-emerald-400":
                status.value === "Succeeded",
              "bg-orange-500 hover:bg-orange-400": status.value === "Preempted",
              "bg-rose-500 hover:bg-rose-200": status.value === "Deleted",
            })}
          ></div>
          <div className="ml-1.5">{status.label}</div>
        </div>
      );
    },
    filterFn: (row, id, value) => {
      return (value as string[]).includes(row.getValue(id));
    },
  },
  {
    accessorKey: "createdAt",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("createdAt")} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("createdAt")}></TableDate>;
    },
    sortingFn: "datetime",
  },
  {
    accessorKey: "startedAt",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={getHeader("startedAt")} />
    ),
    cell: ({ row }) => {
      return <TableDate date={row.getValue("startedAt")}></TableDate>;
    },
    sortingFn: "datetime",
  },
];

export const Component: FC = () => {
  const query = useQuery({
    queryKey: ["overview", "nodes"],
    queryFn: () => apiGetNodes(),
    select: (res) => res.data.data.rows,
  });
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  });

  const data: ClusterNodeInfo[] = useMemo(() => {
    if (!query.data) {
      return [];
    }
    // sort by name
    return query.data
      .sort((a, b) => a.name.localeCompare(b.name))
      .filter((x) => x.role === "worker" && x.isReady)
      .map((x) => {
        const capacity = getAiResource(x.capacity);
        const allocated = getAiResource(x.allocated);
        return {
          name: x.name,
          cpu: {
            percent: (allocated.cpu ?? 0) / (capacity.cpu ?? 1) / 10,
            description: `${((allocated.cpu ?? 0) / 1000).toFixed(1)}/${capacity.cpu ?? 1}`,
          },
          memory: {
            percent:
              ((allocated.memoryNum ?? 0) / (capacity.memoryNum ?? 1)) * 100,
            description: `${(allocated.memoryNum ?? 0).toFixed(1)}/${(capacity.memoryNum ?? 1).toFixed(0)}Gi`,
          },
          gpu: {
            percent: capacity.gpu
              ? ((allocated.gpu ?? 0) / (capacity.gpu ?? 1)) * 100
              : 0,
            description: capacity.gpu
              ? `${(allocated.gpu ?? 0).toFixed(0)}/${capacity.gpu ?? 1}`
              : "",
          },
        };
      });
  }, [query.data]);

  const [jobData, setJobData] = useState<VolcanoJobInfo[]>([]);

  const { data: jobList, isLoading } = useQuery({
    queryKey: ["admin", "tasklist", "volcanoJob"],
    queryFn: () => apiAdminTaskListByType(),
    select: (res) => res.data.data,
  });

  useEffect(() => {
    if (isLoading) return;
    if (!jobList) return;
    const tableData: VolcanoJobInfo[] = jobList
      .filter((job) => !job.isDeleted)
      .map((job) => {
        //const task = convertJTask(t);
        return {
          name: job.name,
          jobName: job.jobName,
          jobType: job.jobType,
          userName: job.userName,
          queue: job.queue,
          status:
            job.isDeleted === true && job.status === "Running"
              ? "Deleted"
              : job.status,
          createdAt: job.createdAt,
          startedAt: job.startedAt,
          deletedAt: job.deletedAt,
        };
      })
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt));

    setJobData(tableData);
  }, [jobList, isLoading]);

  const table = useReactTable({
    data,
    columns,
    state: {
      pagination,
    },
    getCoreRowModel: getCoreRowModel(),
    onPaginationChange: setPagination,
    getPaginationRowModel: getPaginationRowModel(),
  });

  const jobTable = useReactTable({
    data: jobData,
    columns: jobColumns,
    state: {
      pagination,
    },
    getCoreRowModel: getCoreRowModel(),
    onPaginationChange: setPagination,
    getPaginationRowModel: getPaginationRowModel(),
  });

  const updatedAt = new Date(query.dataUpdatedAt).toLocaleString();

  return (
    <>
      <div className="grid auto-rows-max items-start gap-4 md:gap-8 lg:col-span-2">
        <div className="grid gap-4 sm:grid-cols-2 md:grid-cols-4 lg:grid-cols-2 xl:grid-cols-4">
          <Card className="sm:col-span-2">
            <CardHeader className="pb-3">
              <CardTitle>快速开始</CardTitle>
              <CardDescription className="text-balance leading-relaxed">
                在 Crater 启动 Jupyter Lab、打包训练镜像、
                启动深度学习训练任务等。
              </CardDescription>
            </CardHeader>
            <CardFooter>
              <Button>Create New Task</Button>
            </CardFooter>
          </Card>
          <Card className="sm:col-span-2">
            <CardHeader className="pb-3">
              <CardTitle>快速开始</CardTitle>
              <CardDescription className="text-balance leading-relaxed">
                在 Crater 启动 Jupyter Lab、打包训练镜像、
                启动深度学习训练任务等。
              </CardDescription>
            </CardHeader>
            <CardFooter></CardFooter>
          </Card>
        </div>
        <Tabs defaultValue="week">
          <div className="flex items-center">
            <TabsList>
              <TabsTrigger value="week">今日</TabsTrigger>
              <TabsTrigger value="month">本周</TabsTrigger>
              <TabsTrigger value="year">本月</TabsTrigger>
            </TabsList>
            <div className="ml-auto flex items-center gap-2">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 gap-1 text-sm"
                  >
                    <ListFilter className="h-3.5 w-3.5" />
                    <span className="sr-only sm:not-sr-only">Filter</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>Filter by</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuCheckboxItem checked>
                    Running
                  </DropdownMenuCheckboxItem>
                  <DropdownMenuCheckboxItem>Successed</DropdownMenuCheckboxItem>
                  <DropdownMenuCheckboxItem>
                    Traininged
                  </DropdownMenuCheckboxItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <Button size="sm" variant="outline" className="h-7 gap-1 text-sm">
                <File className="h-3.5 w-3.5" />
                <span className="sr-only sm:not-sr-only">Export</span>
              </Button>
            </div>
          </div>
          <TabsContent value="week">
            <Card>
              <CardHeader className="px-7">
                <CardTitle>集群作业</CardTitle>
                <CardDescription>提交的作业数量和状态统计</CardDescription>
              </CardHeader>
              <CardContent>
                <NodeHome
                  table={jobTable}
                  columns={jobColumns}
                  isLoading={isLoading}
                />
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
      <div>
        <Card className="overflow-hidden">
          <CardHeader className="flex flex-row items-start bg-muted/50">
            <div className="grid gap-0.5">
              <CardTitle className="flex items-center gap-2 text-lg">
                集群资源
              </CardTitle>
              <CardDescription>查看各节点的资源使用情况</CardDescription>
            </div>
            <div className="ml-auto flex items-center gap-1">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button size="icon" variant="outline" className="h-8 w-8">
                    <MoreVertical className="h-3.5 w-3.5" />
                    <span className="sr-only">More</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem>Edit</DropdownMenuItem>
                  <DropdownMenuItem>Export</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem>Trash</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </CardHeader>
          <CardContent className="p-6 text-sm">
            <NodeHome
              table={table}
              columns={columns}
              isLoading={query.isLoading}
            />
          </CardContent>
          <CardFooter className="flex flex-row items-center border-t bg-muted/50 px-6 py-3">
            <div className="text-xs text-muted-foreground">
              更新于 <time dateTime="2023-11-23">{updatedAt}</time>
            </div>
            <Pagination className="ml-auto mr-0 w-auto">
              <PaginationContent>
                <PaginationItem>
                  <Button
                    size="icon"
                    variant="outline"
                    className="h-6 w-6"
                    onClick={() => table.previousPage()}
                    disabled={!table.getCanPreviousPage()}
                  >
                    <ChevronLeft className="h-3.5 w-3.5" />
                    <span className="sr-only">Previous Order</span>
                  </Button>
                </PaginationItem>
                <PaginationItem>
                  <Button
                    size="icon"
                    variant="outline"
                    className="h-6 w-6"
                    onClick={() => table.nextPage()}
                    disabled={!table.getCanNextPage()}
                  >
                    <ChevronRight className="h-3.5 w-3.5" />
                    <span className="sr-only">Next Order</span>
                  </Button>
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </CardFooter>
        </Card>
      </div>
    </>
  );
};
