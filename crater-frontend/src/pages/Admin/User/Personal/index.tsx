import {
  DotsHorizontalIcon,
  Pencil2Icon,
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { showErrorToast } from "@/utils/toast";
import { toast } from "sonner";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import {
  apiAdminUserDelete,
  apiAdminUserList,
  apiAdminUserUpdateRole,
} from "@/services/api/admin/user";
import { z } from "zod";
import { useRecoilValue } from "recoil";
import { globalUserInfo } from "@/utils/store";
import { AiResourceSchema, getAiResource } from "@/utils/resource";
import { Badge } from "@/components/ui/badge";
import { QuotaForm } from "./QuotaForm";

const QuotaSchema = z.object({
  hard: AiResourceSchema,
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

export const PersonalUser = () => {
  const [data, setData] = useState<TUser[]>([]);
  const queryClient = useQueryClient();
  const { id: currentUserName } = useRecoilValue(globalUserInfo);

  const { data: userList, isLoading } = useQuery({
    queryKey: ["admin", "userlist"],
    queryFn: apiAdminUserList,
    select: (res) => res.data.data.Users,
  });

  const { mutate: deleteUser } = useMutation({
    mutationFn: (userName: string) => apiAdminUserDelete(userName),
    onSuccess: async (_, userName) => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "userlist"] });
      toast.success(`用户 ${userName} 已删除`);
    },
  });

  const { mutate: updateRole } = useMutation({
    mutationFn: ({ userName, role }: { userName: string; role: string }) =>
      apiAdminUserUpdateRole(userName, role),
    onSuccess: async (_, variables) => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "userlist"] });
      toast.success(`用户 ${variables.userName} 权限已更新`);
    },
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
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
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
            <div className="flex items-center space-x-1">
              <Badge
                className="rounded-sm bg-sidebar-item px-1 font-normal"
                variant={"secondary"}
              >
                {quota.hard.cpu} CPU
              </Badge>
              <Badge
                className="rounded-sm bg-sidebar-item px-1 font-normal"
                variant={"secondary"}
              >
                {quota.hard.gpu} GPU
              </Badge>
              <Badge
                className="rounded-sm bg-sidebar-item px-1 font-normal"
                variant={"secondary"}
              >
                {quota.hard.memoryNum || 0} {quota.hard.memoryUnit}
              </Badge>
            </div>
          );
        },
        // issue: https://github.com/TanStack/table/issues/4591
        sortingFn: (rowA, rowB) => {
          const a = rowA.original.quota.hard.cpu || 0;
          const b = rowB.original.quota.hard.cpu || 0;
          if (a === b) {
            const a = rowA.original.quota.hard.gpu || 0;
            const b = rowB.original.quota.hard.gpu || 0;
            if (a === b) {
              const a = rowA.original.quota.hard.memoryNum || 0;
              const b = rowB.original.quota.hard.memoryNum || 0;
              if (a === b) {
                return 0;
              }
              if (b === 0) return 1;
              if (a === 0) return -1;
              return a - b;
            }
            if (b === 0) return 1;
            if (a === 0) return -1;
            return a - b;
          }
          if (b === 0) return 1;
          if (a === 0) return -1;
          return a - b;
        },
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const user = row.original;
          // eslint-disable-next-line react-hooks/rules-of-hooks
          const [openQuotaDialog, setOpenQuotaDialog] = useState(false);

          return (
            <div>
              <Dialog open={openQuotaDialog} onOpenChange={setOpenQuotaDialog}>
                <DialogTrigger asChild>
                  <Button
                    variant="ghost"
                    className="h-8 w-8 p-0"
                    title="配额管理"
                  >
                    <Pencil2Icon className="h-4 w-4" />
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle className="mb-2">修改配额</DialogTitle>
                    <QuotaForm
                      closeSheet={() => setOpenQuotaDialog(false)}
                      userName={user.userName}
                      quota={user.quota.hard}
                    />
                  </DialogHeader>
                </DialogContent>
              </Dialog>
              <AlertDialog>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      className="h-8 w-8 p-0"
                      title="更多选项"
                    >
                      <DotsHorizontalIcon className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel>操作</DropdownMenuLabel>
                    <DropdownMenuSub>
                      <DropdownMenuSubTrigger>权限</DropdownMenuSubTrigger>
                      <DropdownMenuSubContent>
                        <DropdownMenuRadioGroup value={user.role}>
                          {roles.map((role) => (
                            <DropdownMenuRadioItem
                              key={role.value}
                              value={role.value}
                              onClick={() =>
                                updateRole({
                                  userName: user.userName,
                                  role: role.value,
                                })
                              }
                            >
                              {role.label}
                            </DropdownMenuRadioItem>
                          ))}
                        </DropdownMenuRadioGroup>
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
                    <DropdownMenuSeparator />
                    <AlertDialogTrigger asChild>
                      <DropdownMenuItem>删除</DropdownMenuItem>
                    </AlertDialogTrigger>
                  </DropdownMenuContent>
                </DropdownMenu>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除用户</AlertDialogTitle>
                    <AlertDialogDescription>
                      用户「{user?.userName}」将被删除，请谨慎操作。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        if (user.userName === currentUserName) {
                          showErrorToast(
                            new Error("无法删除自己，如需删除请换个用户登录"),
                          );
                        } else {
                          deleteUser(user.userName);
                        }
                      }}
                    >
                      删除
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          );
        },
      },
    ],
    [deleteUser, currentUserName, updateRole],
  );

  useEffect(() => {
    if (isLoading) return;
    if (!userList) return;
    const tableData: TUser[] = userList.map((user) => ({
      id: user.userID,
      userName: user.userName,
      role: user.role,
      quota: {
        hard: getAiResource(user.quotaHard),
      },
    }));
    setData(tableData);
  }, [userList, isLoading]);

  return (
    <div className="space-y-4">
      <DataTable
        data={data}
        columns={columns}
        toolbarConfig={toolbarConfig}
      ></DataTable>
    </div>
  );
};
