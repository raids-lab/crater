import { Button } from "@/components/ui/button";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import {
  Form,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormControl,
} from "@/components/ui/form";
import FormLabelMust from "@/components/custom/FormLabelMust";
import { DialogTrigger, Dialog, DialogClose } from "@/components/ui/dialog";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import {
  SelectValue,
  SelectTrigger,
  SelectItem,
  SelectContent,
  Select,
  SelectGroup,
} from "@/components/ui/select";
import {
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogCancel,
  AlertDialogAction,
  AlertDialogFooter,
  AlertDialogContent,
  AlertDialog,
  AlertDialogTrigger,
} from "@/components/ui-custom/alert-dialog";
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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import {
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogContent,
} from "@/components/ui/dialog";
import {
  IUserInAccount,
  apiAddUser,
  apiUpdateUser,
  apiRemoveUser,
  apiUserInProjectList,
  apiUserOutOfProjectList,
  Access,
  IUserInAccountCreate,
  apiAccountGet,
} from "@/services/api/account";
import { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/components/custom/DataTable";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { z } from "zod";
import useBreadcrumb from "@/hooks/useBreadcrumb";
import UserRoleBadge, { userRoles } from "@/components/label/UserRoleBadge";
import UserAccessBadge from "@/components/label/UserAccessBadge";
import ResourceBadges from "@/components/label/ResourceBadges";
import UserLabel from "@/components/label/UserLabel";
import { UserRoundPlusIcon } from "lucide-react";
import Quota from "./AccountQuota";
import { toast } from "sonner";

const formSchema = z.object({
  index: z.string().min(1, {
    message: "id不能小于1",
  }),
  role: z.string().min(1, {
    message: "不支持的角色",
  }),
  accessmode: z.string().min(1, {
    message: "不支持的访问权限",
  }),
});

type FormSchema = z.infer<typeof formSchema>;

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "用户名";
    case "role":
      return "账户权限";
    case "accessmode":
      return "读写权限";
    case "capability":
      return "资源上限";
    default:
      return key;
  }
};

const accessmodes = [
  {
    label: "只读",
    value: Access.RO.toString(),
  },
  {
    label: "读写",
    value: Access.RW.toString(),
  },
];

const AccountDetail = () => {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const pid = useMemo(() => Number(id), [id]);

  const setBreadcrumb = useBreadcrumb();

  const accountUsersQuery = useQuery({
    queryKey: ["account", pid, "users"],
    queryFn: () => apiUserInProjectList(pid),
    select: (res) => res.data.data,
  });

  const { data: accountInfo } = useQuery({
    queryKey: ["admin", "accounts", pid],
    queryFn: () => apiAccountGet(pid),
    select: (res) => res.data.data,
    enabled: !!pid,
  });

  // 修改 BreadCrumb
  useEffect(() => {
    if (!accountInfo) return;
    setBreadcrumb([{ title: accountInfo?.nickname }]);
  }, [setBreadcrumb, accountInfo]);

  const { mutate: addUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => apiAddUser(pid, user),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["account", pid, "users"],
      });
    },
  });

  const { mutate: updateUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => apiUpdateUser(pid, user),
    onSuccess: async () => {
      toast.success("用户信息已更新");
      await queryClient.invalidateQueries({
        queryKey: ["account", pid, "users"],
      });
    },
  });

  const [usersOutOfProject, setUsersOutOfProject] = useState<IUserInAccount[]>(
    [],
  );

  const { mutate: deleteUser } = useMutation({
    mutationFn: (user: IUserInAccountCreate) => apiRemoveUser(pid, user),
    onSuccess: async () => {
      toast.success("用户已删除");
      await queryClient.invalidateQueries({
        queryKey: ["account", pid, "users"],
      });
    },
  });

  const { data: usersOutOfProjectData, isLoading: isLoadingUsersOutOfProject } =
    useQuery({
      queryKey: ["usersOutOfProject", pid],
      queryFn: () => apiUserOutOfProjectList(pid),
      select: (res) => res.data.data,
    });

  useEffect(() => {
    if (isLoadingUsersOutOfProject) return;
    if (!usersOutOfProjectData) return;
    setUsersOutOfProject(usersOutOfProjectData);
  }, [usersOutOfProjectData, isLoadingUsersOutOfProject]);

  const columns = useMemo<ColumnDef<IUserInAccount>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <UserLabel attributes={row.original.userInfo} />,
      },
      {
        accessorKey: "role",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("role")} />
        ),
        cell: ({ row }) => {
          return <UserRoleBadge role={row.getValue("role")} />;
        },
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "accessmode",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("accessmode")}
          />
        ),
        cell: ({ row }) => (
          <UserAccessBadge access={row.getValue("accessmode")} />
        ),
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id));
        },
      },
      {
        accessorKey: "capability",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("capability")}
          />
        ),
        cell: ({ row }) => {
          return <ResourceBadges resources={row.original.quota.capability} />;
        },
        enableSorting: false,
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const user = row.original;
          return (
            <div>
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
                      <DropdownMenuSubTrigger>角色</DropdownMenuSubTrigger>
                      <DropdownMenuSubContent>
                        <DropdownMenuRadioGroup value={`${user.role}`}>
                          {userRoles.map((role) => (
                            <DropdownMenuRadioItem
                              key={role.value}
                              value={`${role.value}`}
                              onClick={() =>
                                updateUser({
                                  id: user.id,
                                  name: user.name,
                                  role: role.value,
                                  accessmode: user.accessmode,
                                })
                              }
                            >
                              {role.label}
                            </DropdownMenuRadioItem>
                          ))}
                        </DropdownMenuRadioGroup>
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
                    <DropdownMenuSub>
                      <DropdownMenuSubTrigger>权限</DropdownMenuSubTrigger>
                      <DropdownMenuSubContent>
                        <DropdownMenuRadioGroup value={`${user.accessmode}`}>
                          {accessmodes.map((accessmode) => (
                            <DropdownMenuRadioItem
                              key={accessmode.value}
                              value={`${accessmode.value}`}
                              onClick={() =>
                                updateUser({
                                  id: user.id,
                                  name: user.name,
                                  role: user.role,
                                  accessmode: accessmode.value,
                                })
                              }
                            >
                              {accessmode.label}
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
                      用户「{user?.name}」将被删除，请谨慎操作。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        deleteUser(user);
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
    [deleteUser, updateUser],
  );

  const toolbarConfig: DataTableToolbarConfig = {
    filterInput: {
      placeholder: "搜索用户名",
      key: "name",
    },
    filterOptions: [
      {
        key: "role",
        title: "角色",
        option: userRoles,
      },
      {
        key: "accessmode",
        title: "权限",
        option: accessmodes,
      },
    ],
    getHeader: getHeader,
  };

  const [openSheet, setOpenSheet] = useState(false);

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      index: "0",
      role: "0",
      accessmode: "0",
    },
  });

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    addUser({
      id: usersOutOfProject.at(parseInt(values.index))?.id as number,
      name: usersOutOfProject.at(parseInt(values.index))?.name as string,
      role: values.role,
      accessmode: values.accessmode,
    });
    setOpenSheet(false);
  };

  return (
    <div className="grid gap-6 lg:grid-cols-4">
      <DataTable
        query={accountUsersQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
      >
        <Dialog open={openSheet} onOpenChange={setOpenSheet}>
          <DialogTrigger asChild>
            <Button className="h-8">
              <UserRoundPlusIcon className="h-4 w-4" />
              添加用户
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>添加用户</DialogTitle>
            </DialogHeader>
            <Form {...form}>
              <form
                onSubmit={form.handleSubmit(onSubmit)}
                className="space-y-4"
              >
                <FormField
                  control={form.control}
                  name="index"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        用户
                        <FormLabelMust />
                      </FormLabel>

                      <Select
                        onValueChange={field.onChange}
                        defaultValue={field.value}
                      >
                        <FormControl>
                          <SelectTrigger className="w-full" id="name">
                            <SelectValue placeholder="Select users" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectGroup>
                            {usersOutOfProject.length > 0 &&
                              usersOutOfProject.map((user, index) => (
                                <SelectItem
                                  key={index}
                                  value={index.toString()}
                                >
                                  {user.name}
                                </SelectItem>
                              ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        可选择一位或多位用户加入到账户中
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="role"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        角色
                        <FormLabelMust />
                      </FormLabel>
                      <Select
                        onValueChange={field.onChange}
                        defaultValue={field.value.toString()}
                      >
                        <FormControl>
                          <SelectTrigger className="w-full" id="role">
                            <SelectValue placeholder="Select users" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="2">普通用户</SelectItem>
                            <SelectItem value="3">管理员</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>配置用户在账户中的角色</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="accessmode"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        访问权限
                        <FormLabelMust />
                      </FormLabel>
                      <Select
                        onValueChange={field.onChange}
                        defaultValue={field.value.toString()}
                      >
                        <FormControl>
                          <SelectTrigger className="w-full" id="accessmode">
                            <SelectValue placeholder="Select users" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="2">只读</SelectItem>
                            <SelectItem value="3">读写</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      <FormDescription>
                        设置用户在账户空间的访问权限
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <DialogFooter>
                  <DialogClose asChild>
                    <Button variant="outline">取消</Button>
                  </DialogClose>
                  <Button type="submit">添加</Button>
                </DialogFooter>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </DataTable>
      <div className="flex flex-col gap-3 lg:pt-10">
        <Quota accountID={pid} />
      </div>
    </div>
  );
};

export default AccountDetail;
