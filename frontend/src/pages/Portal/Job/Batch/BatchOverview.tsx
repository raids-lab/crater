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
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { Link, useNavigate } from "react-router-dom";
import { TableDate } from "@/components/custom/TableDate";
import { toast } from "sonner";
import { getHeader, jobToolbarConfig } from "@/pages/Portal/Job/statuses";
import { logger } from "@/utils/loglevel";
import Quota from "../Interactive/Quota";
import JobPhaseLabel from "@/components/custom/JobPhaseLabel";
import {
  Card,
  CardDescription,
  CardFooter,
  CardHeader,
} from "@/components/ui/card";
import { CardTitle } from "@/components/ui-custom/card";
import SplitButton from "@/components/custom/SplitButton";
import { IJobInfo, JobType } from "@/services/api/vcjob";
import { REFETCH_INTERVAL } from "@/config/task";
import NodeBadges from "@/components/custom/NodeBadges";
import ResourceBadges from "@/components/custom/ResourceBadges";
import JobTypeLabel from "@/components/custom/JobTypeLabel";
import { globalJobUrl } from "@/utils/store";
import { useAtom } from "jotai";

export const Component = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const jobType = useAtom(globalJobUrl)[0];

  const batchQuery = useQuery({
    queryKey: ["job", "batch"],
    queryFn: apiJobBatchList,
    select: (res) =>
      res.data.data.filter((task) => task.jobType !== JobType.Jupyter),
    refetchInterval: REFETCH_INTERVAL,
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

  const batchColumns = useMemo<ColumnDef<IJobInfo>[]>(
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
        accessorKey: "jobType",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("jobType")} />
        ),
        cell: ({ row }) => (
          <JobTypeLabel jobType={row.getValue<JobType>("jobType")} />
        ),
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
        accessorKey: "owner",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("owner")} />
        ),
        cell: ({ row }) => <div>{row.getValue("owner")}</div>,
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
        accessorKey: "nodes",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("nodes")} />
        ),
        cell: ({ row }) => {
          const nodes = row.getValue<string[]>("nodes");
          return <NodeBadges nodes={nodes} />;
        },
      },
      {
        accessorKey: "resources",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("resources")}
          />
        ),
        cell: ({ row }) => {
          const resources = row.getValue<Record<string, string>>("resources");
          return <ResourceBadges resources={resources} />;
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
      <div className="grid gap-5 lg:col-span-3 lg:grid-cols-4">
        <Card className="row-span-2 flex flex-col justify-between lg:col-span-2">
          <CardHeader>
            <CardTitle>批处理作业</CardTitle>
            <CardDescription className="text-balance pt-2 leading-relaxed">
              指无须人工干预而执行系列程序的作业，包含单机作业、Pytorch
              分布式训练作业、 Tensorflow 分布式训练作业、Ray 分布式训练作业、
              OpenMPI 分布式计算作业等。
            </CardDescription>
          </CardHeader>
          <CardFooter>
            <SplitButton
              title="batch"
              urls={[
                {
                  url: "portal/job/batch/new-tensorflow",
                  name: " Tensorflow 作业",
                },
                {
                  url: "portal/job/batch/new-pytorch",
                  name: " Pytorch 作业",
                },
                {
                  url: "portal/job/batch/new-ray",
                  name: " Ray 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-deepspeed",
                  name: " DeepSpeed 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-openmpi",
                  name: " OpenMPI 作业",
                  disabled: true,
                },
                {
                  url: "portal/job/batch/new-" + jobType,
                  name: "自定义作业（单机）",
                },
              ]}
            />
          </CardFooter>
        </Card>
        <Quota />
      </div>
      <DataTable
        query={batchQuery}
        columns={batchColumns}
        toolbarConfig={jobToolbarConfig}
      ></DataTable>
    </>
  );
};
