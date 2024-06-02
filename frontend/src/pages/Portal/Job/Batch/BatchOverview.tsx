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
} from "@/components/ui-custom/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiJobDelete, JobPhase, apiJobBatchList } from "@/services/api/vcjob";

import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { Link, useNavigate } from "react-router-dom";
import { TableDate } from "@/components/custom/TableDate";
import { toast } from "sonner";
import { getHeader } from "@/pages/Portal/Job/Batch/statuses";
import { logger } from "@/utils/loglevel";
import Quota from "../Interactive/Quota";
import JobPhaseLabel, { jobPhases } from "@/components/custom/JobPhaseLabel";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
} from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import SplitButton from "@/components/custom/SplitButton";
import { IVolcanoJobInfo } from "@/services/api/admin/task";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索作业名称",
    key: "name",
  },
  filterOptions: [
    {
      key: "status",
      title: "作业状态",
      option: jobPhases,
    },
  ],
  getHeader: getHeader,
};

export const Component = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const batchQuery = useQuery({
    queryKey: ["job", "batch"],
    queryFn: apiJobBatchList,
    select: (res) => res.data.data.filter((task) => task.jobType !== "jupyter"),
  });

  const refetchTaskList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["job"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "quota"] }),
        queryClient.invalidateQueries({ queryKey: ["aitask", "stats"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };

  const { mutate: deleteTask } = useMutation({
    mutationFn: apiJobDelete,
    onSuccess: async () => {
      await refetchTaskList();
      toast.success("作业已删除");
    },
  });

  const batchColumns = useMemo<ColumnDef<IVolcanoJobInfo>[]>(
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
          />
        ),
        cell: ({ row }) => (
          <div>
            <Checkbox
              checked={row.getIsSelected()}
              onCheckedChange={(value) => row.toggleSelected(!!value)}
              aria-label="Select row"
            />
          </div>
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
          <Link
            to={row.original.jobName}
            className="underline-offset-4 hover:underline"
          >
            {row.getValue("name")}
          </Link>
        ),
      },
      {
        accessorKey: "status",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("status")} />
        ),
        cell: ({ row }) => {
          return <JobPhaseLabel jobPhase={row.getValue<JobPhase>("status")} />;
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
          return <TableDate date={row.getValue("createdAt")}></TableDate>;
        },
        sortingFn: "datetime",
      },
      {
        accessorKey: "startedAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("startedAt")}
          />
        ),
        cell: ({ row }) => {
          return <TableDate date={row.getValue("startedAt")}></TableDate>;
        },
        sortingFn: "datetime",
      },
      {
        accessorKey: "completedAt",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("completedAt")}
          />
        ),
        cell: ({ row }) => {
          return <TableDate date={row.getValue("completedAt")}></TableDate>;
        },
        sortingFn: "datetime",
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
                    onClick={() => navigate(`${taskInfo.jobName}`)}
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
                  <AlertDialogTitle>删除作业</AlertDialogTitle>
                  <AlertDialogDescription>
                    作业「{taskInfo?.name}」将不再可见，请谨慎操作。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => deleteTask(taskInfo.jobName)}
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

  return (
    <>
      <div className="col-span-3 grid gap-4 md:grid-cols-2 md:gap-6 xl:grid-cols-4">
        <Card className="flex flex-col justify-between lg:col-span-2">
          <CardHeader>
            <CardTitle>批处理作业</CardTitle>
            <CardDescription className="text-balance pt-2 leading-relaxed">
              指无须人工干预而执行系列程序的作业，包含单机作业、Pytorch
              分布式训练作业、 Tensorflow 分布式训练作业、Ray 分布式训练作业、
              OpenMPI 分布式计算作业等。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <SplitButton
              title="batch"
              urls={[
                { url: "training", name: "单机作业" },
                { url: "tensorflow", name: " Tensorflow 作业" },
                { url: "pytorch", name: " Pytorch 作业", disabled: true },
                { url: "ray", name: " Ray 作业", disabled: true },
                { url: "openmpi", name: " OpenMPI 作业", disabled: true },
              ]}
            />
          </CardContent>
        </Card>
        <Quota />
      </div>
      <DataTable
        query={batchQuery}
        columns={batchColumns}
        toolbarConfig={toolbarConfig}
      ></DataTable>
    </>
  );
};
