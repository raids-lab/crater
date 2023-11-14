import {
  DotsHorizontalIcon,
  StarFilledIcon,
  StarIcon,
} from "@radix-ui/react-icons";
import { ColumnDef } from "@tanstack/react-table";

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
import { apiAiTaskDelete } from "@/services/api/aiTask";
import { logger } from "@/utils/loglevel";
import { showErrorToast } from "@/utils/toast";
import { useToast } from "@/components/ui/use-toast";
import { DataTable } from "@/components/DataTable";
import { DataTableColumnHeader } from "@/components/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/DataTable/DataTableToolbar";
import { apiAdminUserList } from "@/services/api/admin/user";
import { z } from "zod";

const ResourceSchema = z
  .object({
    cpu: z.number(),
    gpu: z.number(),
    memory: z.number(),
  })
  .strict();

const QuotaSchema = z.object({
  hard: ResourceSchema,
  hardUsed: ResourceSchema,
});

const TUserSchema = z.object({
  id: z.number(),
  userName: z.string(),
  role: z.string(),
  quota: QuotaSchema,
});

type TUser = z.infer<typeof TUserSchema>;

const getHeader = (key: string): string => {
  switch (key) {
    case "userName":
      return "用户名";
    case "role":
      return "权限";
    case "quota":
      return "配额";
    default:
      return key;
  }
};

const roles = [
  {
    label: "管理员",
    value: "admin",
    icon: StarFilledIcon,
  },
  {
    label: "普通用户",
    value: "user",
    icon: StarIcon,
  },
];

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索用户名",
    key: "userName",
  },
  filterOptions: [
    {
      key: "role",
      title: "权限",
      option: roles,
    },
  ],
  getHeader: getHeader,
};

export function Component() {
  const [data, setData] = useState<TUser[]>([]);
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const { data: userList, isLoading } = useQuery({
    queryKey: ["admin", "userlist"],
    retry: 1,
    queryFn: apiAdminUserList,
    select: (res) => res.data.data.Users,
    onSuccess: (data) => {
      logger.debug("Data is: ", data);
    },
    onError: (err) => showErrorToast("获取任务列表失败", err),
  });

  const { mutate: deleteTask } = useMutation({
    mutationFn: (id: number) => apiAiTaskDelete(id),
    onSuccess: () => {
      queryClient
        .invalidateQueries({ queryKey: ["tasklist"] })
        .then(() => {
          toast({
            title: `删除成功`,
            description: `任务已删除`,
          });
        })
        .catch((err) => {
          showErrorToast("刷新任务列表失败", err);
        });
    },
    onError: (err) => showErrorToast("无法删除任务", err),
  });

  const columns = useMemo<ColumnDef<TUser>[]>(
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
        accessorKey: "userName",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("userName")}
          />
        ),
        cell: ({ row }) => <div>{row.getValue("userName")}</div>,
      },
      {
        accessorKey: "role",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("role")} />
        ),
        cell: ({ row }) => {
          const role = roles.find(
            (role) => role.value === row.getValue("role"),
          );
          if (!role) {
            return null;
          }
          return (
            <div className="flex items-center">
              {role.icon && (
                <role.icon className="mr-2 h-4 w-4 text-muted-foreground" />
              )}
              <span>{role.label}</span>
            </div>
          );
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        id: "quota",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("quota")} />
        ),
        cell: ({ row }) => {
          const { quota } = row.original;
          return (
            <div className="flex items-center">
              <span className="mr-2">CPU:{quota.hard.cpu}</span>
              <span className="mr-2">GPU:{quota.hard.gpu}</span>
              <span className="mr-2">MEM:{quota.hard.memory}</span>
            </div>
          );
        },
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const taskInfo = row.original;

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
                <DropdownMenuItem
                  onClick={() => {
                    // check if browser support clipboard
                    deleteTask(taskInfo.id);
                  }}
                >
                  删除
                </DropdownMenuItem>
                {/* <DropdownMenuSeparator /> */}
                {/* <DropdownMenuItem>View customer</DropdownMenuItem>
              <DropdownMenuItem>View payment details</DropdownMenuItem> */}
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [deleteTask],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!userList) return;
    const tableData: TUser[] = userList.map((user) => ({
      id: user.userID,
      userName: user.userName,
      role: user.role,
      quota: {
        hard: {
          cpu: 10,
          gpu: 10,
          memory: 40,
        },
        hardUsed: {
          cpu: 1,
          gpu: 2,
          memory: 3,
        },
      },
    }));
    setData(tableData);
  }, [userList, isLoading]);

  return (
    <div className="space-y-4 px-6 py-4">
      <DataTable
        data={data}
        columns={columns}
        toolbarConfig={toolbarConfig}
      ></DataTable>
    </div>
  );
}
