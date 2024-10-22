import { DataTableColumnHeader } from "@/components/custom/PagenationDataTable/DataTableColumnHeader";
import {
  IProject,
  apiAdminAccountList,
  apiProjectCreate,
  apiProjectUpdate,
} from "@/services/api/account";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import {
  PencilIcon,
  PlusCircleIcon,
  TrashIcon,
  UserRoundIcon,
} from "lucide-react";
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
import { useMemo } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { toast } from "sonner";
import { apiProjectDelete } from "@/services/api/account";
import { useNavigate } from "react-router-dom";
import { useAtomValue } from "jotai";
import { globalSettings } from "@/utils/store";
import { DataTable } from "@/components/custom/DataTable";
import ResourceBadges from "@/components/label/ResourceBadges";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { apiResourceList } from "@/services/api/resource";
import { ProjectSheet } from "./AccountForm";

function convertResourcesToMap(
  resources: Resource[],
  field: keyof Resource,
): {
  [key: string]: string;
} {
  const resourceMap: { [key: string]: string } = {};
  resources.forEach((resource) => {
    if (resource[field] !== "") {
      resourceMap[resource.name] = resource[field] as string;
    }
  });
  return resourceMap;
}

const resourceSchema = z.array(
  z.object({
    name: z.string().min(1, {
      message: "资源名称不能为空",
    }),
    guaranteed: z
      .union([
        z.string().min(0, {
          message: "资源不能为空",
        }),
        z.number().int().min(0, {
          message: "资源至少为0",
        }),
      ])
      .default("null"),
    deserved: z
      .union([
        z.string().min(0, {
          message: "资源不能为空",
        }),
        z.number().int().min(0, {
          message: "资源至少为0",
        }),
      ])
      .default("null"),
    capacity: z
      .union([
        z.string().min(0, {
          message: "资源不能为空",
        }),
        z.number().int().min(0, {
          message: "资源至少为0",
        }),
      ])
      .default("null"),
  }),
);

const formSchema = z.object({
  name: z
    .string()
    .min(1, {
      message: "账户名称不能为空",
    })
    .max(16, {
      message: "账户名称最多16个字符",
    }),
  resources: resourceSchema,
  expiredAt: z.date().optional(),
  admins: z.array(z.string()),
});

type FormSchema = z.infer<typeof formSchema>;

interface Resource {
  name: string;
  guaranteed: number | string;
  deserved: number | string;
  capacity: number | string;
}

const getHeader = (key: string): string => {
  switch (key) {
    case "nickname":
      return "账户名称";
    case "deserved":
      return "应得";
    case "guaranteed":
      return "保证";
    case "capacity":
      return "上限";
    default:
      return key;
  }
};

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    key: "name",
    placeholder: "搜索账户名称",
  },
  filterOptions: [],
  getHeader: getHeader,
};

export const Account = () => {
  const navigate = useNavigate();

  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["admin", "accounts"],
    queryFn: () => apiAdminAccountList(),
    select: (res) => res.data.data,
  });

  const { mutate: deleteProject } = useMutation({
    mutationFn: (projectId: string) => apiProjectDelete({ id: projectId }),
    onSuccess: async (_, projectName) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "accounts"],
      });
      toast.success(`账户 ${projectName} 已删除`);
    },
  });

  const { scheduler } = useAtomValue(globalSettings);
  const { mutate: updateProject, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiProjectUpdate({
        name: values.name,
        guaranteed: convertResourcesToMap(values.resources, "guaranteed"),
        deserved: convertResourcesToMap(values.resources, "deserved"),
        capacity: convertResourcesToMap(values.resources, "capacity"),
        expiredAt: values.expiredAt,
        withoutVolcano: scheduler !== "volcano",
      }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "accounts"],
      });
      toast.success(`账户 ${name} 更新成功`);
    },
  });

  const columns = useMemo<ColumnDef<IProject>[]>(
    () => [
      {
        accessorKey: "nickname",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("nickname")}
          />
        ),
        cell: ({ row }) => <div>{row.getValue("nickname")}</div>,
        enableSorting: false,
      },
      {
        accessorKey: "guaranteed",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("guaranteed")}
          />
        ),
        cell: ({ row }) => {
          const quota = row.getValue<Record<string, string>>("guaranteed");
          return <ResourceBadges resources={quota} />;
        },
        enableSorting: false,
      },
      {
        accessorKey: "deserved",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("deserved")}
          />
        ),
        cell: ({ row }) => {
          const quota = row.getValue<Record<string, string>>("deserved");
          return <ResourceBadges resources={quota} />;
        },
        enableSorting: false,
      },
      {
        accessorKey: "capacity",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("capacity")}
          />
        ),
        cell: ({ row }) => {
          const quota = row.getValue<Record<string, string>>("capacity");
          return <ResourceBadges resources={quota} />;
        },
        enableSorting: false,
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          const proj = row.original;
          const defaultValues: FormSchema = {
            name: proj.nickname,
            resources: [],
            expiredAt: undefined,
            admins: [],
          };
          const nameRecord = {} as Record<string, Resource>;
          Object.entries(proj.guaranteed).map(([key, value]) =>
            nameRecord[key]
              ? (nameRecord[key].guaranteed = value)
              : (nameRecord[key] = {
                  name: key,
                  guaranteed: value,
                  deserved: "",
                  capacity: "",
                } as Resource),
          );
          Object.entries(proj.deserved).map(([key, value]) =>
            nameRecord[key]
              ? (nameRecord[key].deserved = value)
              : (nameRecord[key] = {
                  name: key,
                  deserved: value,
                  guaranteed: "",
                  capacity: "",
                } as Resource),
          );
          Object.entries(proj.capacity).map(([key, value]) =>
            nameRecord[key]
              ? (nameRecord[key].capacity = value)
              : (nameRecord[key] = {
                  name: key,
                  capacity: value,
                  guaranteed: "",
                  deserved: "",
                } as Resource),
          );
          const resourcesData = Object.entries(nameRecord).map(
            ([, value]) => value,
          );

          return (
            <div className="flex flex-row items-center justify-center gap-1">
              <Button
                title="管理用户"
                onClick={() => navigate(`${proj.id}`)}
                variant="outline"
                className="h-8 w-8"
                size="icon"
              >
                <UserRoundIcon className="h-4 w-4" />
              </Button>
              <ProjectSheet
                formData={defaultValues}
                isPending={isPending}
                resourcesData={resourcesData}
                apiProjcet={updateProject}
              >
                <Button
                  title="修改配额"
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                >
                  <PencilIcon className="h-4 w-4" />
                </Button>
              </ProjectSheet>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    title="删除账户"
                    variant="outline"
                    size="icon"
                    className="h-8 w-8"
                  >
                    <TrashIcon className="h-4 w-4" />
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除账户</AlertDialogTitle>
                    <AlertDialogDescription>
                      账户 {proj?.nickname} 将被删除，请谨慎操作。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={() => {
                        deleteProject(String(proj.id));
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
    [deleteProject, isPending, navigate, updateProject],
  );

  const defaultValues: FormSchema = {
    name: "",
    resources: [
      {
        name: "cpu",
        guaranteed: "",
        deserved: "",
        capacity: "",
      },
      {
        name: "memory",
        guaranteed: "",
        deserved: "",
        capacity: "",
      },
    ],
    expiredAt: undefined,
    admins: [],
  };

  const { data: resourcesData } = useQuery({
    queryKey: ["resources", "list"],
    queryFn: () => apiResourceList(true),
    select: (res) => {
      return res.data.data.map(
        (item) =>
          ({
            name: item.name,
            guaranteed: "",
            deserved: "",
            capacity: "",
          }) as Resource,
      );
    },
  });

  const { mutate: createNewProject, isPending: isCreatePending } = useMutation({
    mutationFn: (values: z.infer<typeof formSchema>) =>
      apiProjectCreate({
        name: values.name,
        guaranteed: convertResourcesToMap(values.resources, "guaranteed"),
        deserved: convertResourcesToMap(values.resources, "deserved"),
        capacity: convertResourcesToMap(values.resources, "capacity"),
        expiredAt: values.expiredAt,
        withoutVolcano: scheduler !== "volcano",
      }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ["admin", "accounts"],
      });
      toast.success(`账户 ${name} 创建成功`);
    },
  });

  return (
    <DataTable query={query} columns={columns} toolbarConfig={toolbarConfig}>
      <ProjectSheet
        formData={defaultValues}
        isPending={isCreatePending}
        resourcesData={resourcesData}
        apiProjcet={createNewProject}
      >
        <Button className="h-8">
          <PlusCircleIcon className="-ml-0.5 mr-2 h-4 w-4" />
          新建账户
        </Button>
      </ProjectSheet>
    </DataTable>
  );
};
