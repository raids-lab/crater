import {
  DotsHorizontalIcon,
  StarFilledIcon,
  StarIcon,
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
import { showErrorToast } from "@/utils/toast";
import { useToast } from "@/components/ui/use-toast";
import { DataTable } from "@/components/DataTable";
import { DataTableColumnHeader } from "@/components/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/DataTable/DataTableToolbar";
import {
  apiAdminUserDelete,
  apiAdminUserList,
} from "@/services/api/admin/user";
import { z } from "zod";
import { useRecoilValue } from "recoil";
import { globalUserInfo } from "@/utils/store";
import { ResourceSchema, getResource } from "@/utils/resource";
import { Badge } from "@/components/ui/badge";

const QuotaSchema = z.object({
  hard: ResourceSchema,
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
  const { id: currentUserName } = useRecoilValue(globalUserInfo);

  const { data: userList, isLoading } = useQuery({
    queryKey: ["admin", "userlist"],
    retry: 1,
    queryFn: apiAdminUserList,
    select: (res) => res.data.data.Users,
    onError: (err) => showErrorToast("获取用户列表失败", err),
  });

  const { mutate: deleteUser } = useMutation({
    mutationFn: (userName: string) => apiAdminUserDelete(userName),
    onSuccess: (_, userName) => {
      queryClient
        .invalidateQueries({ queryKey: ["admin", "userlist"] })
        .then(() => {
          toast({
            title: `删除成功`,
            description: `用户 ${userName} 已删除`,
          });
        })
        .catch((err) => {
          showErrorToast("刷新用户列表失败", err);
        });
    },
    onError: (err) => showErrorToast("无法删除用户", err),
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
        accessorFn: (row) => row.quota,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("quota")} />
        ),
        cell: ({ row }) => {
          const { quota } = row.original;
          return (
            <div className="flex items-center">
              <Badge
                className="mr-2"
                variant={
                  quota.hard.cpu && quota.hard.cpu > 0 ? "default" : "secondary"
                }
              >
                CPU {quota.hard.cpu}
              </Badge>
              <Badge
                className="mr-2"
                variant={
                  quota.hard.gpu && quota.hard.gpu > 0 ? "default" : "secondary"
                }
              >
                GPU {quota.hard.gpu}
              </Badge>
              <Badge
                className="mr-2"
                variant={
                  quota.hard.gpu && quota.hard.gpu > 0 ? "default" : "secondary"
                }
              >
                MEM {quota.hard.memory}
              </Badge>
            </div>
          );
        },
        // issue: https://github.com/TanStack/table/issues/4591
        sortingFn: (rowA, rowB) => {
          const a = rowA.original.quota.hard.cpu || 0;
          const b = rowB.original.quota.hard.cpu || 0;
          if (a === b) return 0;
          if (b === 0) return 1;
          if (a === 0) return -1;
          return b - a;
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
                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>删除</DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除用户</AlertDialogTitle>
                  <AlertDialogDescription>
                    用户「{taskInfo?.userName}」将被删除，请谨慎操作。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={() => {
                      if (taskInfo.userName === currentUserName) {
                        showErrorToast(
                          "删除失败",
                          new Error("无法删除自己，如需删除请换个用户登录"),
                        );
                      } else {
                        deleteUser(taskInfo.userName);
                      }
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
    [deleteUser, currentUserName],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!userList) return;
    const tableData: TUser[] = userList.map((user) => ({
      id: user.userID,
      userName: user.userName,
      role: user.role,
      quota: {
        hard: getResource(user.quotaHard),
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
