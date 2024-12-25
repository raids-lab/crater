import type { FC } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useState, useEffect, useMemo, useRef } from "react";
import { DataTable } from "@/components/custom/DataTable";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { FileSizeComponent } from "@/components/file/FileSize";
import { DownloadIcon, PlusIcon, ArrowLeftIcon } from "@radix-ui/react-icons";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { File, Folder, Trash2 } from "lucide-react";
import { useSetAtom } from "jotai";
import { globalBreadCrumb } from "@/utils/store";
import {
  FileItem,
  apiFileDelete,
  apiGetQueueFiles,
  apiMkdir,
} from "@/services/api/file";
import { ACCESS_TOKEN_KEY } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";
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
import FileUpload from "@/components/file/FileUpload";

export const Component: FC = () => {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [dirName, setDirName] = useState<string>("");
  const setBreadcrumb = useSetAtom(globalBreadCrumb);
  const path = useMemo(
    () => pathname.replace(/^\/admin\/data/, ""),
    [pathname],
  );

  useEffect(() => {
    const pathParts = pathname.split("/").filter(Boolean);
    const breadcrumb = pathParts.map((value, index) => {
      value = decodeURIComponent(value);
      if (index == 0 && value == "admin") {
        return {
          title: "Admin",
        };
      } else if (index == 1 && value == "data") {
        return {
          title: "数据管理",
          path: "/admin/data",
          isEmpty: true,
        };
      } else if (index == 2 && value == "account") {
        return {
          title: "账户文件",
          path: "/admin/data/account",
        };
      }
      return {
        title: value,
        path: `/${pathParts.slice(0, index + 1).join("/")}`,
      };
    });
    // 删除第一个元素
    breadcrumb.shift();
    // 将最后一个元素的 path 设置为 undefined
    if (breadcrumb.length > 1) {
      breadcrumb[breadcrumb.length - 1].path = undefined;
    }
    setBreadcrumb(breadcrumb);
  }, [pathname, setBreadcrumb]);

  const backpath = useMemo(() => {
    // pathname is /xxx/xxx/xxx/aaa
    // backpath is /xxx/xxx/xxx
    return pathname.replace(/\/[^/]+$/, "");
  }, [pathname]);
  const data = useQuery({
    queryKey: ["data", "accountfiles", path],
    queryFn: () => apiGetQueueFiles(`${path}`),
    select: (res) => res.data.data ?? [],
  });
  const { mutate: deleteFile } = useMutation({
    mutationFn: (req: string) => apiFileDelete(req),
    onSuccess: async () => {
      toast.success("删除成功");
      await queryClient.invalidateQueries({
        queryKey: ["data", "accountfiles", path],
      });
    },
  });
  const columns = useMemo<ColumnDef<FileItem>[]>(
    () => [
      {
        accessorKey: "name",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("name")} />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return (
              <div style={{ display: "flex", alignItems: "center" }}>
                <Folder className="mr-2 size-5 text-yellow-600 dark:text-yellow-400" />
                <Button
                  onClick={() => {
                    // setFilepath(filepath + "/" + row.original.name);
                    navigate(pathname + "/" + row.original.name);
                  }}
                  variant={"link"}
                  className="h-8 px-0 text-left font-normal text-secondary-foreground"
                >
                  {row.getValue("name")}
                </Button>
              </div>
            );
          } else {
            return (
              <div style={{ display: "flex", alignItems: "center" }}>
                <File className="mr-2 size-5 text-muted-foreground" />
                <span className="text-secondary-foreground">
                  {row.getValue("name")}
                </span>
              </div>
            );
          }
        },
        enableSorting: false,
      },
      {
        accessorKey: "modifytime",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("modifytime")}
          />
        ),
        cell: ({ row }) => {
          return (
            <TimeDistance date={row.getValue("modifytime")}></TimeDistance>
          );
        },
        //sortingFn: "datetime",
        enableSorting: false,
      },
      {
        accessorKey: "size",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("size")} />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return;
          } else {
            return <FileSizeComponent size={row.getValue("size")} />;
          }
        },
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          if (!row.original.isdir) {
            return (
              <div className="flex flex-row space-x-1">
                <Button
                  variant="outline"
                  className="h-8 w-8 p-0 hover:text-sky-700"
                  title="下载文件"
                  onClick={() => {
                    const link = `${import.meta.env.VITE_API_BASE_URL}ss/download${path}/${row.original.name}`;
                    const o = new XMLHttpRequest();
                    o.open("GET", link);
                    o.responseType = "blob";
                    const token = localStorage.getItem(ACCESS_TOKEN_KEY);
                    o.setRequestHeader("Authorization", "Bearer " + token);
                    o.onload = function () {
                      if (o.status == 200) {
                        const content = o.response as string;
                        const a = document.createElement("a");
                        a.style.display = "none";
                        a.download = row.original.name || "";
                        const blob = new Blob([content]);
                        a.href = URL.createObjectURL(blob);
                        document.body.appendChild(a);
                        a.click();
                        document.body.removeChild(a);
                        toast.success("下载文件成功！");
                      } else {
                        toast.error("下载失败：" + o.statusText);
                      }
                    };
                    o.send();
                    toast.info("正在下载该文件");
                  }}
                >
                  <DownloadIcon className="size-4" />
                </Button>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <div>
                      <Button
                        variant="outline"
                        className="h-8 w-8 p-0 hover:text-destructive"
                        title="删除文件"
                      >
                        <Trash2 size={16} strokeWidth={4} />
                      </Button>
                    </div>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>删除文件</AlertDialogTitle>
                      <AlertDialogDescription>
                        文件将删除，请小心不要误删文件
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>取消</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
                        onClick={() => {
                          deleteFile(path + "/" + row.original.name);
                        }}
                      >
                        删除
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            );
          } else {
            return (
              <div className="flex flex-row space-x-1">
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <div>
                      <Button
                        variant="outline"
                        className="h-8 w-8 p-0 hover:text-destructive"
                        title="删除文件夹"
                      >
                        <Trash2 size={16} strokeWidth={2} />
                      </Button>
                    </div>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>删除文件夹</AlertDialogTitle>
                      <AlertDialogDescription>
                        文件将删除，请小心不要误删文件
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>取消</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
                        onClick={() => {
                          deleteFile(path + "/" + row.original.name);
                        }}
                      >
                        删除
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            );
          }
        },
      },
    ],
    [navigate, deleteFile, path, pathname],
  );

  const getHeader = (key: string): string => {
    switch (key) {
      case "name":
        return "文件名";
      case "modifytime":
        return "更新于";
      case "size":
        return "文件大小";
      default:
        return key;
    }
  };
  const toolbarConfig: DataTableToolbarConfig = {
    filterInput: {
      placeholder: "搜索名称",
      key: "name",
    },
    filterOptions: [],
    getHeader: getHeader,
  };

  const refInput2 = useRef<HTMLInputElement>(null);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setDirName(e.target.value);
  };
  const queryClient = useQueryClient();

  const clientCreateDir = async (path: string) => {
    //console.log("dirName:" + dirName);
    if (dirName != "") {
      await apiMkdir(`${path}/${dirName}`)
        .then(() => {
          toast.success("文件夹已创建");
        })
        .catch((error) => {
          showErrorToast(error);
        });
    }
  };
  const { mutate: CreateDir } = useMutation({
    mutationFn: () => clientCreateDir(path),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["data", "accountfiles", path],
      });
    },
  });

  return (
    <DataTable
      query={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      className="col-span-3"
    >
      <Button
        variant="outline"
        size="icon"
        onClick={() => {
          // setFilepath(backfilepath);
          navigate(backpath);
        }}
        className="h-8 w-8"
        disabled={pathname === "/admin/data/account"}
      >
        <ArrowLeftIcon />
      </Button>
      <FileUpload
        uploadPath={path}
        disabled={pathname === "/admin/data/account"}
      ></FileUpload>
      <Dialog>
        <DialogTrigger asChild>
          <Button
            variant="outline"
            className="h-8 w-8"
            size="icon"
            disabled={pathname === "/admin/data/account"}
          >
            <PlusIcon />
          </Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>创建新文件夹</DialogTitle>
            <DialogDescription>输入文件夹名</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="name" className="text-right">
                名称
              </Label>
              <Input
                id="name"
                type="text"
                defaultValue=""
                className="col-span-3"
                ref={refInput2}
                onChange={handleChange}
              />
            </div>
          </div>
          <DialogClose>
            <DialogFooter>
              <Button type="submit" onClick={() => CreateDir()}>
                创建文件夹
              </Button>
            </DialogFooter>
          </DialogClose>
        </DialogContent>
      </Dialog>
    </DataTable>
  );
};
