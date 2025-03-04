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
import {
  ArrowLeftIcon,
  DownloadIcon,
  File,
  Folder,
  FolderPlusIcon,
  Globe,
  Trash2,
} from "lucide-react";
import { useSetAtom } from "jotai";
import { globalBreadCrumb } from "@/utils/store";
import {
  FileItem,
  MoveFile,
  apiFileDelete,
  apiGetFiles,
  apiMkdir,
  apiMoveFile,
} from "@/services/api/file";
import { ACCESS_TOKEN_KEY } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";
import { getFolderTitle } from "@/components/file/LazyFileTree";
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
import FolderNavigation from "./FolderNavigation";

const getHeader = (key: string): string => {
  switch (key) {
    case "name":
      return "名称";
    case "modifytime":
      return "更新于";
    case "size":
      return "大小";
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

export const Component: FC = () => {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [dirName, setDirName] = useState<string>("");
  const setBreadcrumb = useSetAtom(globalBreadCrumb);

  const path = useMemo(
    () => pathname.replace(/^\/portal\/data\/filesystem/, ""),
    [pathname],
  );

  useEffect(() => {
    const pathParts = pathname.split("/").filter(Boolean);
    const breadcrumb = pathParts.map((value, index) => {
      value = decodeURIComponent(value);
      if (index == 0 && value == "portal") {
        return {
          title: "Portal",
        };
      } else if (index == 1 && value == "data") {
        return {
          title: "数据管理",
          path: "/portal/data",
          isEmpty: true,
        };
      } else if (index == 2 && value == "filesystem") {
        return {
          title: "文件系统",
          path: "/portal/data/filesystem",
        };
      } else if (index == 3) {
        if (value == "user" || value == "account") {
          return {
            title: getFolderTitle(value),
            path: `/portal/data/filesystem/${value}`,
            isEmpty: true,
          };
        } else {
          return {
            title: getFolderTitle(value),
            path: `/portal/data/filesystem/${value}`,
          };
        }
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
    return pathname.replace(/\/[^/]+$/, "");
  }, [pathname]);

  const query = useQuery({
    queryKey: ["data", "filesystem", path],
    queryFn: () => apiGetFiles(`${path}`),
    select: (res) => {
      return (
        res.data.data
          ?.map((r) => {
            return {
              name: r.name,
              modifytime: r.modifytime,
              isdir: r.isdir,
              size: r.size,
              sys: r.sys,
            };
          })
          .sort((a, b) => {
            if (a.isdir && !b.isdir) {
              return -1; // a在b之前
            } else if (!a.isdir && b.isdir) {
              return 1; // a在b之后
            } else {
              return a.name.localeCompare(b.name);
            }
          }) ?? []
      );
    },
  });

  const isRoot = useMemo(
    () => pathname === "/portal/data/filesystem",
    [pathname],
  );
  const { mutate: deleteFile } = useMutation({
    mutationFn: (req: string) => apiFileDelete(req),
    onSuccess: async () => {
      toast.success("删除成功");
      await queryClient.invalidateQueries({
        queryKey: ["data", "filesystem", path],
      });
    },
  });

  const { mutate: moveFile } = useMutation({
    mutationFn: ({ fileData, path }: { fileData: MoveFile; path: string }) =>
      apiMoveFile(fileData, path),
    onSuccess: async () => {
      toast.success("文件移动成功");
      await queryClient.invalidateQueries({
        queryKey: ["data", "filesystem", path],
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
          return (
            <>
              {path.startsWith("/public") ? (
                <>
                  {row.original.isdir ? null : (
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
                          o.setRequestHeader(
                            "Authorization",
                            "Bearer " + token,
                          );
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
                    </div>
                  )}
                </>
              ) : (
                <>
                  {row.original.isdir ? (
                    <div className="flex flex-row space-x-1">
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <div>
                            <Button
                              variant="outline"
                              className="h-8 w-8 p-0 hover:text-destructive"
                              title="删除文件夹"
                              disabled={isRoot}
                            >
                              <Trash2 size={16} strokeWidth={2} />
                            </Button>
                          </div>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>删除文件夹</AlertDialogTitle>
                            <AlertDialogDescription>
                              文件夹{row.original.name}
                              将删除，请确认是否删除该文件夹
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
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <div>
                            <Button
                              variant="outline"
                              className="h-8 w-8 p-0 hover:text-destructive"
                              title="移动文件"
                            >
                              <Globe size={16} strokeWidth={2} />
                            </Button>
                          </div>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>选择移动位置</AlertDialogTitle>
                            <AlertDialogDescription>
                              是否将文件夹{row.original.name}
                              移动到公共空间或当前账户空间？
                              请注意，如果进行移动，当前位置下文件夹会不再存在。
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>取消</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => {
                                moveFile({
                                  fileData: {
                                    type: 1,
                                    fileName: row.original.name,
                                  },
                                  path: `${path}/${row.original.name}`,
                                });
                              }}
                            >
                              移动到公共空间
                            </AlertDialogAction>
                            <AlertDialogAction
                              onClick={() => {
                                moveFile({
                                  fileData: {
                                    type: 2,
                                    fileName: row.original.name,
                                  },
                                  path: `${path}/${row.original.name}`,
                                });
                              }}
                            >
                              移动到账户空间
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  ) : (
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
                          o.setRequestHeader(
                            "Authorization",
                            "Bearer " + token,
                          );
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
                              <Trash2 size={16} strokeWidth={2} />
                            </Button>
                          </div>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>删除文件</AlertDialogTitle>
                            <AlertDialogDescription>
                              文件{row.original.name}
                              将删除，请确认是否删除该文件
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
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <div>
                            <Button
                              variant="outline"
                              className="h-8 w-8 p-0 hover:text-destructive"
                              title="公开文件"
                            >
                              <Globe size={16} strokeWidth={2} />
                            </Button>
                          </div>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>选择移动位置</AlertDialogTitle>
                            <AlertDialogDescription>
                              是否将文件{row.original.name}
                              移动到公共空间或当前账户空间？
                              请注意，如果进行移动，当前位置下文件会不再存在。
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>取消</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => {
                                moveFile({
                                  fileData: {
                                    type: 1,
                                    fileName: row.original.name,
                                  },
                                  path: `${path}/${row.original.name}`,
                                });
                              }}
                            >
                              移动到公共空间
                            </AlertDialogAction>
                            <AlertDialogAction
                              onClick={() => {
                                moveFile({
                                  fileData: {
                                    type: 2,
                                    fileName: row.original.name,
                                  },
                                  path: `${path}/${row.original.name}`,
                                });
                              }}
                            >
                              移动到账户空间
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  )}
                </>
              )}
            </>
          );
        },
      },
    ],
    [navigate, pathname, path, isRoot, deleteFile, moveFile],
  );

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
        queryKey: ["data", "filesystem", path],
      });
    },
  });

  const handleReturnNavigation = (backpath: string) => {
    if (backpath == "/portal/data/filesystem/user") {
      navigate("/portal/data/filesystem");
    } else if (backpath == "/portal/data/filesystem/account") {
      navigate("/portal/data/filesystem");
    } else {
      navigate(backpath);
    }
  };

  return (
    <>
      {isRoot ? (
        <FolderNavigation data={query.data}></FolderNavigation>
      ) : (
        <DataTable
          query={query}
          columns={columns}
          toolbarConfig={toolbarConfig}
        >
          <Button
            variant="outline"
            size="icon"
            onClick={() => {
              handleReturnNavigation(backpath);
            }}
            className="h-8 w-8"
            disabled={isRoot}
            title="返回上一级"
          >
            <ArrowLeftIcon className="size-4" />
          </Button>
          <FileUpload uploadPath={path} disabled={isRoot}></FileUpload>
          <Dialog>
            <DialogTrigger asChild>
              <Button
                variant="outline"
                className="h-8 w-8"
                size="icon"
                disabled={isRoot}
                title="创建文件夹"
              >
                <FolderPlusIcon className="size-4" />
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
      )}
    </>
  );
};
