import { ColumnDef } from "@tanstack/react-table";
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiJobDelete, JobPhase, apiJobBatchList } from "@/services/api/vcjob";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { toast } from "sonner";
import { getHeader, jobToolbarConfig } from "@/pages/Portal/Job/statuses";
import { logger } from "@/utils/loglevel";
import JobPhaseLabel from "@/components/badge/JobPhaseBadge";
import SplitLinkButton from "@/components/button/SplitLinkButton";
import { IJobInfo, JobType } from "@/services/api/vcjob";
import { REFETCH_INTERVAL } from "@/config/task";
import NodeBadges from "@/components/badge/NodeBadges";
import ResourceBadges from "@/components/badge/ResourceBadges";
import JobTypeLabel from "@/components/badge/JobTypeBadge";
import { globalJobUrl } from "@/utils/store";
import { useAtomValue } from "jotai";
import { Trash2Icon } from "lucide-react";
import DocsButton from "@/components/button/DocsButton";
import { JobNameCell } from "@/components/label/JobNameLabel";
import { JobActionsMenu } from "@/components/job/JobActionsMenu";

const VolcanoOverview = () => {
  const queryClient = useQueryClient();
  const jobType = useAtomValue(globalJobUrl);

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
        cell: ({ row }) => <JobNameCell jobInfo={row.original} />,
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
          const resources = row.getValue<Record<string, string> | undefined>(
            "resources",
          );
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
          return <TimeDistance date={row.getValue("createdAt")}></TimeDistance>;
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
          return <TimeDistance date={row.getValue("startedAt")}></TimeDistance>;
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
          return (
            <TimeDistance date={row.getValue("completedAt")}></TimeDistance>
          );
        },
        sortingFn: "datetime",
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const jobInfo = row.original;
          return <JobActionsMenu jobInfo={jobInfo} onDelete={deleteTask} />;
        },
      },
    ],
    [deleteTask],
  );

  return (
    <DataTable
      info={{
        title: "批处理作业",
        description: "提交无须人工干预而执行系列程序的作业",
      }}
      storageKey="portal_batch_job_overview"
      query={batchQuery}
      columns={batchColumns}
      toolbarConfig={jobToolbarConfig}
      multipleHandlers={[
        {
          title: (rows) => `停止或删除 ${rows.length} 个作业`,
          description: (rows) => (
            <>
              作业 {rows.map((row) => row.original.name).join(", ")}{" "}
              将被停止或删除，确认要继续吗？
            </>
          ),
          icon: <Trash2Icon className="text-destructive" />,
          handleSubmit: (rows) => {
            rows.forEach((row) => {
              deleteTask(row.original.jobName);
            });
          },
          isDanger: true,
        },
      ]}
    >
      <div className="flex flex-row gap-3">
        <DocsButton title="查看文档" url="quick-start/batchprocess" />
        <SplitLinkButton
          title="batch"
          urls={[
            {
              url: "portal/job/batch/new-" + jobType,
              name: "自定义作业（单机）",
            },
            {
              url: "portal/job/batch/new-tensorflow",
              name: " Tensorflow 作业",
              disabled: jobType === "spjobs",
            },
            {
              url: "portal/job/batch/new-pytorch",
              name: " Pytorch 作业",
              disabled: jobType === "spjobs",
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
          ]}
        />
      </div>
    </DataTable>
  );
};

export default VolcanoOverview;
