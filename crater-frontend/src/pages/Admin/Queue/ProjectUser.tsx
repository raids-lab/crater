import { Button } from "@/components/ui/button";
import {
  DotsHorizontalIcon,
  StarFilledIcon,
  StarIcon,
} from "@radix-ui/react-icons";
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
  User,
  apiAddUser,
  apiUpdateUser,
  apiRemoveUser,
  apiUserInProjectList,
  apiUserOutOfProjectList,
  Access,
} from "@/services/api/project";
import { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/components/custom/OldDataTable";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
import { Role } from "@/services/api/auth";
import { z } from "zod";
import useBreadcrumb from "@/hooks/useDetailBreadcrumb";

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

const roles = [
  {
    label: "普通用户",
    value: Role.User.toString(),
    icon: StarIcon,
  },
  {
    label: "管理员",
    value: Role.Admin.toString(),
    icon: StarFilledIcon,
  },
];

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "用户名";
    case "role":
      return "权限";
    case "accessmode":
      return "读写权限";
    default:
      return key;
  }
};

const accessmodes = [
  {
    label: "只读",
    value: Access.RO.toString(),
    icon: StarIcon,
  },
  {
    label: "读写",
    value: Access.RW.toString(),
    icon: StarFilledIcon,
  },
];

const UserProjectManagement = () => {
  const { id } = useParams();

  const setBreadcrumb = useBreadcrumb();

  // 修改 BreadCrumb
  useEffect(() => {
    setBreadcrumb([{ title: `${id}` ?? "" }]);
  }, [setBreadcrumb, id]);

  const queryClient = useQueryClient();
  const pid = useMemo(() => Number(id), [id]);

  const [usersInProject, setUsersInProject] = useState<User[]>([]);

  const { mutate: addUser } = useMutation({
    mutationFn: (user: User) => apiAddUser(pid, user),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["usersInProject", pid],
      });
    },
  });

  const { mutate: updateUser } = useMutation({
    mutationFn: (user: User) => apiUpdateUser(pid, user),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["usersInProject", pid],
      });
    },
  });

  const { data: usersInProjectData, isLoading } = useQuery({
    queryKey: ["usersInProject", pid],
    queryFn: () => apiUserInProjectList(pid),
    select: (res) => res.data.data,
    refetchInterval: 5000,
  });

  useEffect(() => {
    if (isLoading) return;
    if (!usersInProjectData) return;
    const tableData: User[] = usersInProjectData;
    setUsersInProject(tableData);
  }, [usersInProjectData, isLoading]);

  const [usersOutOfProject, setUsersOutOfProject] = useState<User[]>([]);

  const { mutate: deleteUser } = useMutation({
    mutationFn: (user: User) => apiRemoveUser(pid, user),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["usersOutOfProject", pid],
      });
    },
  });

  const { data: usersOutOfProjectData, isLoading: isLoadingUsersOutOfProject } =
    useQuery({
      queryKey: ["usersOutOfProject", pid],
      queryFn: () => apiUserOutOfProjectList(pid),
      select: (res) => res.data.data,
      refetchInterval: 5000,
    });

  useEffect(() => {
    if (isLoadingUsersOutOfProject) return;
    if (!usersOutOfProjectData) return;
    setUsersOutOfProject(usersOutOfProjectData);
  }, [usersOutOfProjectData, isLoadingUsersOutOfProject]);

  const columns = useMemo<ColumnDef<User>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => <div>{row.getValue("name")}</div>,
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
        accessorKey: "accessmode",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("accessmode")}
          />
        ),
        cell: ({ row }) => {
          const status = accessmodes.find(
            (status) => status.value === row.getValue("accessmode"),
          );
          if (!status) {
            return null;
          }
          return (
            <div className="flex items-center">
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
                          {roles.map((role) => (
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
        option: roles,
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
    <>
      <DataTable
        data={usersInProject}
        columns={columns}
        toolbarConfig={toolbarConfig}
        loading={isLoading}
      >
        <Dialog open={openSheet} onOpenChange={setOpenSheet}>
          <DialogTrigger asChild>
            <Button className="h-8">添加用户</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>添加用户</DialogTitle>
            </DialogHeader>
            <Form {...form}>
              <form
                // eslint-disable-next-line @typescript-eslint/no-misused-promises
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
    </>
  );
};

export default UserProjectManagement;
