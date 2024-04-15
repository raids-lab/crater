import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { apiAdminProjectList } from "@/services/api/admin/user";
import { ProjectStatus } from "@/services/api/project";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import {
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  PaginationState,
  SortingState,
  VisibilityState,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  DataTableToolbarConfig,
  DataTableToolbarRight,
} from "@/components/custom/DataTable/DataTableToolbar";
import DataTableCard from "@/components/custom/DataTable/DataTableCard";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface DataTableProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  setStatus: (status: ProjectStatus) => void;
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  toolbarConfig: DataTableToolbarConfig;
  loading: boolean;
  onPaginationChange: OnChangeFn<PaginationState>;
  rowCount?: number;
  pagination: PaginationState;
  className?: string;
}

function TableWithTabs<TData, TValue>({
  setStatus,
  columns,
  data,
  toolbarConfig,
  loading,
  onPaginationChange,
  rowCount,
  pagination,
}: DataTableProps<TData, TValue>) {
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [sorting, setSorting] = useState<SortingState>([]);

  const [activeTab, setActiveTab] = useState<string>("active");
  const [cachedRowCount, setCachedRowCount] = useState(rowCount);

  useEffect(() => {
    if (rowCount && rowCount !== cachedRowCount) setCachedRowCount(rowCount);
  }, [rowCount, cachedRowCount]);

  useEffect(() => {
    if (activeTab === "active") {
      setStatus(ProjectStatus.Active);
    } else if (activeTab === "pending") {
      setStatus(ProjectStatus.Pending);
    } else if (activeTab === "inactive") {
      setStatus(ProjectStatus.Inactive);
    }
  }, [activeTab, setStatus]);

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnVisibility,
      columnFilters,
      pagination,
    },
    // pagination
    manualPagination: true,
    onPaginationChange,
    rowCount: cachedRowCount,
    // shadcn example props
    enableRowSelection: true,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  return (
    <div className="flex flex-col gap-2 lg:col-span-3">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="flex flex-row items-center justify-between gap-2">
          <div>
            <TabsList>
              <TabsTrigger value="pending">待审批</TabsTrigger>
              <TabsTrigger value="active">活跃中</TabsTrigger>
              <TabsTrigger value="inactive">已停用</TabsTrigger>
            </TabsList>
          </div>
          <DataTableToolbarRight table={table} config={toolbarConfig} />
        </div>
      </Tabs>
      <DataTableCard table={table} loading={loading} columns={columns} />
    </div>
  );
}

type Project = {
  id: number;
  name: string;
  status: ProjectStatus;
  quota: {
    cpu: number;
    cpuReq: number;
    gpu: number;
    gpuMem: number;
    gpuMemReq: number;
    gpuReq: number;
    job: number;
    jobReq: number;
    mem: number;
    memReq: number;
    node: number;
    nodeReq: number;
    storage: number;
    extra: string;
  };
};

const getHeader = (key: string): string => {
  switch (key) {
    case "id":
      return "ID";
    case "name":
      return "项目名称";
    case "quota":
      return "配额";
    default:
      return key;
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  searchKey: "name",
  filterOptions: [],
  getHeader: getHeader,
};

export const Project = ({ isPersonal }: { isPersonal: boolean }) => {
  const [pagination, setPagination] = useState<PaginationState>({
    pageSize: 10,
    pageIndex: 0,
  });

  const [data, setData] = useState<Project[]>([]);
  const [status, setStatus] = useState<ProjectStatus>(ProjectStatus.Active);
  const { data: projects, isLoading } = useQuery({
    queryKey: [
      "admin",
      "projects",
      isPersonal,
      status,
      pagination.pageIndex,
      pagination.pageSize,
    ],
    queryFn: () =>
      apiAdminProjectList({
        pageIndex: pagination.pageIndex,
        pageSize: pagination.pageSize,
        isPersonal,
        status,
      }),
    select: (res) => res.data.data,
  });

  const columns = useMemo<ColumnDef<Project>[]>(
    () => [
      {
        accessorKey: "id",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("id")} />
        ),
        cell: ({ row }) => <div>{row.getValue("id")}</div>,
        enableSorting: false,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
        enableSorting: false,
      },
      {
        accessorKey: "quota",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("quota")} />
        ),
        cell: ({ row }) => {
          const quota = row.getValue<Project["quota"]>("quota");
          return (
            <div className="grid grid-cols-3 rounded-md border p-3 text-xs md:grid-cols-6">
              <div>节点: {quota.node === -1 ? "~" : quota.node}</div>
              <div>任务: {quota.job === -1 ? "~" : quota.job}</div>
              <div>CPU: {quota.cpu}</div>
              <div>GPU: {quota.gpu}</div>
              <div>内存: {quota.mem}GB</div>
              <div>存储: {quota.storage}GB</div>
            </div>
          );
        },
        enableSorting: false,
      },
    ],
    [],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!projects?.rows) return;
    const tableData: Project[] = projects.rows
      //.filter((task) => !task.isDeleted)
      .map((p) => {
        return {
          id: p.ID,
          name: p.name,
          status: p.status,
          quota: {
            cpu: p.cpu,
            cpuReq: p.cpuReq,
            gpu: p.gpu,
            gpuMem: p.gpuMem,
            gpuMemReq: p.gpuMemReq,
            gpuReq: p.gpuReq,
            job: p.job,
            jobReq: p.jobReq,
            mem: p.mem,
            memReq: p.memReq,
            node: p.node,
            nodeReq: p.nodeReq,
            storage: p.storage,
            extra: p.extra,
          },
        };
      });
    setData(tableData);
  }, [projects, isLoading]);

  return (
    <TableWithTabs
      data={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      setStatus={setStatus}
      loading={isLoading}
      pagination={pagination}
      onPaginationChange={setPagination}
      rowCount={projects?.count || 0}
    />
  );
};

export default function AdminProjectPersonal() {
  return <Project isPersonal={true} />;
}
