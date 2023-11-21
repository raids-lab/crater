import {
  CheckCircledIcon,
  CrossCircledIcon,
  CircleIcon,
  ClockIcon,
  DotsHorizontalIcon,
  StopwatchIcon,
} from "@radix-ui/react-icons";
import { ColumnDef } from "@tanstack/react-table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useToast } from "@/components/ui/use-toast";
import { DataTable } from "@/components/DataTable";
import { DataTableColumnHeader } from "@/components/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/DataTable/DataTableToolbar";
import { useSetRecoilState } from "recoil";
import { globalBreadCrumb } from "@/utils/store";
import { useNavigate, useRoutes } from "react-router-dom";
import DlJobDetail from "./Detail";
import {
  apiDlTaskDelete,
  apiDlTaskList,
} from "@/services/api/recommend/dlTask";

type TaskInfo = {
  id: number;
  name: string;
  status: string;
  createdAt: string;
};

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "任务名称";
    case "status":
      return "状态";
    case "createdAt":
      return "创建时间";
    default:
      return key;
  }
};

// Initial/Pending/Running/Finished/Failed
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
    value: "Finished",
    label: "Finished",
    icon: CheckCircledIcon,
  },
];

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索任务名称",
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

const AiJobHome = () => {
  const [data, setData] = useState<TaskInfo[]>([]);
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const navigate = useNavigate();

  const { data: taskList, isLoading } = useQuery({
    queryKey: ["dltask", "list"],
    queryFn: apiDlTaskList,
    select: (res) => res.data.data,
  });

  const { mutate: deleteTask } = useMutation({
    mutationFn: (name: string) => apiDlTaskDelete(name),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["dltask", "list"] });
      toast({
        title: "删除成功",
        description: "任务已删除",
      });
    },
  });

  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  useEffect(() => {
    setBreadcrumb([
      {
        title: "深度推荐训练任务",
      },
    ]);
  }, [setBreadcrumb]);

  const columns = useMemo<ColumnDef<TaskInfo>[]>(
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
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
            className="ml-2"
          />
        ),
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => (
          <Button
            onClick={() => navigate(`${row.original.name}`)}
            variant={"link"}
            className="h-8 px-0 font-normal text-secondary-foreground"
          >
            {row.getValue("name")}
          </Button>
        ),
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
          const taskInfo = row.original;

          return (
            <AlertDialog>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">操作</span>
                    <DotsHorizontalIcon className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>操作</DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => navigate(`${taskInfo.name}`)}
                  >
                    详情
                  </DropdownMenuItem>
                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>删除</DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除任务</AlertDialogTitle>
                  <AlertDialogDescription>
                    任务「{taskInfo?.name}」将不再可见，请谨慎操作。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => {
                      // check if browser support clipboard
                      deleteTask(taskInfo.name);
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          );
        },
      },
    ],
    [deleteTask, navigate],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!taskList) return;
    const tableData: TaskInfo[] = taskList.map((task, index) => ({
      id: index,
      name: task.name,
      status: task.status.phase,
      createdAt: task.creationTimestamp,
    }));
    setData(tableData);
  }, [taskList, isLoading]);

  return (
    <div className="space-y-4 px-6 py-4">
      <DataTable data={data} columns={columns} toolbarConfig={toolbarConfig}>
        <Button className="h-8 min-w-fit">新建任务</Button>
      </DataTable>
    </div>
  );
};

export const Component = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <AiJobHome />,
    },
    {
      path: ":name",
      element: <DlJobDetail />,
    },
  ]);

  return <>{routes}</>;
};
