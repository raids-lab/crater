import { DataTable } from "@/components/DataTable";
import { DataTableColumnHeader } from "@/components/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/DataTable/DataTableToolbar";
import { Card, CardContent } from "@/components/ui/card";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { apiDlTaskInfo, apiDlTaskPods } from "@/services/api/recommend/dlTask";
import { globalBreadCrumb } from "@/utils/store";
import { Checkbox } from "@/components/ui/checkbox";
import {
  CheckCircledIcon,
  CircleIcon,
  ClockIcon,
  CrossCircledIcon,
  DotsHorizontalIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";
import { useQuery } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { useEffect, type FC, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useSetRecoilState } from "recoil";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";

type PodInfo = {
  id: string;
  name: string;
  status: string;
  createdAt: string;
};

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "Pod 名称";
    case "status":
      return "状态";
    case "createdAt":
      return "创建时间";
    default:
      return key;
  }
};

const statuses = [
  {
    value: "Initial",
    label: "Initial",
    icon: CircleIcon,
  },
  {
    value: "Pending",
    label: "Pending",
    icon: ClockIcon,
  },
  {
    value: "Running",
    label: "Running",
    icon: StopwatchIcon,
  },
  {
    value: "Failed",
    label: "Failed",
    icon: CrossCircledIcon,
  },
  {
    value: "Succeeded",
    label: "Succeeded",
    icon: CheckCircledIcon,
  },
];

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索 Pod 名称",
    key: "name",
  },
  filterOptions: [
    {
      key: "status",
      title: "状态",
      option: statuses,
    },
  ],
  getHeader: getHeader,
};

// route format: /portal/job/ai/detail?id=xxx
const DlJobDetail: FC = () => {
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  const navigate = useNavigate();
  const { name: taskName } = useParams();

  const taskInfo = useQuery({
    queryKey: ["dltask", "task", taskName],
    queryFn: () => apiDlTaskInfo(taskName ?? ""),
    select: (res) => res.data.data,
    enabled: !!taskName,
  });

  const podInfo = useQuery({
    queryKey: ["dltask", "task", taskName, "pod"],
    queryFn: () => apiDlTaskPods(taskName ?? ""),
    select: (res) => res.data.data,
    enabled: !!taskName,
  });

  useEffect(() => {
    if (taskInfo.isLoading) {
      return;
    }
    setBreadcrumb([
      {
        title: "深度推荐训练任务",
        path: "/recommend/job/dl",
      },
      {
        title: `任务「${taskInfo.data?.name}」详情`,
      },
    ]);
  }, [setBreadcrumb, taskInfo]);

  const columns = useMemo<ColumnDef<PodInfo>[]>(
    () => [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
            aria-label="Select all"
            className="ml-2"
          />
        ),
        cell: ({ row }) => (
          <>
            <Checkbox
              checked={row.getIsSelected()}
              onCheckedChange={(value) => row.toggleSelected(!!value)}
              aria-label="Select row"
              className="ml-2"
            />
          </>
        ),
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
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
            <div className="flex w-[100px] items-center">
              {status.icon && (
                <status.icon className="mr-2 h-4 w-4 text-muted-foreground" />
              )}
              <span>{status.label}</span>
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
          <DataTableColumnHeader
            column={column}
            title={getHeader("createdAt")}
          />
        ),
        cell: ({ row }) => {
          // row format: "2023-10-30T03:21:03.733Z"
          const date = new Date(row.getValue("createdAt"));
          const formatted = date.toLocaleString("zh-CN", {
            timeZone: "Asia/Shanghai",
          });
          return <div>{formatted}</div>;
        },
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const podInfo = row.original;
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
                <DropdownMenuItem onClick={() => navigate(`${podInfo.name}`)}>
                  详情
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [],
  );

  const data: PodInfo[] = useMemo(() => {
    if (!podInfo.data) {
      return [];
    }
    return podInfo.data.map((pod) => ({
      id: pod.metadata.uid,
      name: pod.metadata.name,
      status: pod.status.phase,
      createdAt: pod.metadata.creationTimestamp,
    }));
  }, [podInfo.data]);

  if (taskInfo.isLoading || podInfo.isLoading) {
    return <></>;
  }

  return (
    <div className="space-y-4 px-6 py-4">
      <DataTable data={data} columns={columns} toolbarConfig={toolbarConfig} />
      <Card>
        <ScrollArea className="w-full">
          <CardContent className="w-full max-w-screen-sm pt-6">
            <pre className="text-clip text-sm">
              {JSON.stringify(taskInfo.data, null, 2)}
            </pre>
          </CardContent>
          <ScrollBar orientation="horizontal" className="hidden" />
        </ScrollArea>
      </Card>
    </div>
  );
};

export default DlJobDetail;
