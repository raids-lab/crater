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
import { useAtomValue, useSetAtom } from "jotai";
import { globalAccount, globalBreadCrumb } from "@/utils/store";
import {
  FileItem,
  MoveFile,
  apiFileDelete,
  apiMkdir,
  apiMoveFile,
} from "@/services/api/file";
import { ACCESS_TOKEN_KEY } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";
import {
  getAdminFolderTitle,
  getFolderTitle,
} from "@/components/file/LazyFileTree";
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
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import { FileSelectDialog } from "./FileSelectDialog";
import TooltipButton from "../custom/TooltipButton";
import { asyncUrlApiBaseAtom } from "@/utils/store/config";
import { AccessMode } from "@/services/api/auth";

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

interface FilesystemTableProps {
  apiGetFiles: (
    path: string,
  ) => Promise<AxiosResponse<IResponse<FileItem[] | undefined>>>;
  isadmin: boolean;
}

const FileActions = ({
  apiBaseURL,
  deleteFile,
  moveFile,
  isDir,
  name,
  path,
  canShow,
}: {
  apiBaseURL: string;
  deleteFile: (path: string) => void;
  moveFile: ({ fileData, path }: { fileData: MoveFile; path: string }) => void;
  isDir: boolean;
  name: string;
  path: string;
  canShow: boolean;
}) => {
  const [isMoveDialogOpen, setIsMoveDialogOpen] = useState(false);
  return (
    <div className="flex flex-row space-x-1">
      {/* 下载按钮（仅文件显示）*/}
      {!isDir && (
        <TooltipButton
          variant="outline"
          className="h-8 w-8 p-0 hover:text-sky-700"
          tooltipContent="下载文件"
          onClick={() => {
            const link = `${apiBaseURL}ss/download${path}/${name}`;
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
                a.download = name || "";
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
        </TooltipButton>
      )}

      {/* 删除操作 （有读写权限显示）*/}
      {canShow && (
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <div>
              <TooltipButton
                className="hover:text-destructive h-8 w-8 p-0"
                variant="outline"
                size="icon"
                tooltipContent={`删除${isDir ? "文件夹" : "文件"}`}
              >
                <Trash2 size={16} strokeWidth={2} />
              </TooltipButton>
            </div>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                删除{isDir ? "文件夹" : "文件"}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {isDir ? "文件夹" : "文件"}
                {name}将被永久删除
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={() => {
                  deleteFile(path + "/" + name);
                }}
              >
                删除
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}

      {/* 移动操作 （有读写权限显示）*/}
      {canShow && (
        <AlertDialog open={isMoveDialogOpen} onOpenChange={setIsMoveDialogOpen}>
          <AlertDialogTrigger asChild>
            <div>
              <TooltipButton
                className="hover:text-destructive h-8 w-8 p-0"
                variant="outline"
                size="icon"
                tooltipContent={`移动${isDir ? "文件夹" : "文件"}`}
              >
                <Globe size={16} strokeWidth={2} />
              </TooltipButton>
            </div>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle className="text-lg">
                移动{isDir ? "文件夹" : "文件"}
              </AlertDialogTitle>
              <AlertDialogDescription>
                <div className="text-foreground font-medium">
                  正在移动：
                  <span className="text-primary">{name}</span>
                </div>
                <p className="text-sm">
                  请选择目标位置，移动后原位置将不再保留
                </p>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction asChild>
                <FileSelectDialog
                  value=""
                  handleSubmit={(item) => {
                    setIsMoveDialogOpen(false);
                    moveFile({
                      fileData: {
                        fileName: name,
                        dst: item.id + "/" + name,
                      },
                      path: `${path}/${name}`,
                    });
                  }}
                  isrw={true}
                  disabled={false}
                  allowSelectFile={!isDir}
                />
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </div>
  );
};

export function FileSystemTable({
  apiGetFiles,
  isadmin,
}: FilesystemTableProps) {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [dirName, setDirName] = useState<string>("");
  const setBreadcrumb = useSetAtom(globalBreadCrumb);
  const apiBaseURL = useAtomValue(asyncUrlApiBaseAtom);

  const path = useMemo(() => {
    const basePattern = isadmin
      ? /^\/admin\/data\/filesystem/
      : /^\/portal\/data\/filesystem/;
    return pathname.replace(basePattern, "");
  }, [pathname, isadmin]);

  useEffect(() => {
    const pathParts = pathname.split("/").filter(Boolean);
    const breadcrumb = pathParts.map((value, index) => {
      value = decodeURIComponent(value);
      if (!isadmin) {
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
          return {
            title: getFolderTitle(value),
            path: `/portal/data/filesystem/${value}`,
          };
        }
      } else {
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
        } else if (index == 2 && value == "filesystem") {
          return {
            title: "文件系统",
            path: "/admin/data/filesystem",
          };
        } else if (index == 3) {
          return {
            title: getAdminFolderTitle(value),
            path: `/admin/data/filesystem/${value}`,
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
  }, [isadmin, pathname, setBreadcrumb]);

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

  const isRoot = useMemo(() => {
    const basePath = isadmin
      ? "/admin/data/filesystem"
      : "/portal/data/filesystem";
    return pathname === basePath;
  }, [pathname, isadmin]);
  const { mutate: deleteFile } = useMutation({
    mutationFn: (req: string) => apiFileDelete(req),
    onSuccess: async () => {
      toast.success("删除成功");
      await queryClient.invalidateQueries({
        queryKey: ["data", "filesystem", path],
      });
    },
  });

  const token = useAtomValue(globalAccount);
  const canShow = useMemo(() => {
    if (path.startsWith("/public")) {
      return token.accessPublic === AccessMode.ReadWrite;
    }
    if (path.startsWith("/account")) {
      return token.accessQueue === AccessMode.ReadWrite; // 根据实际权限规则调整
    }
    if (path.startsWith("/user") || path.startsWith("/admin")) {
      return true;
    }
    return false; // 默认不显示
  }, [path, token.accessPublic, token.accessQueue]);

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
                  className="text-secondary-foreground h-8 px-0 text-left font-normal"
                >
                  {row.getValue("name")}
                </Button>
              </div>
            );
          } else {
            return (
              <div style={{ display: "flex", alignItems: "center" }}>
                <File className="text-muted-foreground mr-2 size-5" />
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
          const { isdir, name } = row.original;

          return (
            <FileActions
              apiBaseURL={apiBaseURL}
              deleteFile={deleteFile}
              moveFile={moveFile}
              isDir={isdir}
              name={name}
              path={path}
              canShow={canShow}
            />
          );
        },
      },
    ],
    [navigate, pathname, apiBaseURL, deleteFile, moveFile, path, canShow],
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
    navigate(backpath);
  };

  return (
    <>
      {isRoot ? (
        <FolderNavigation
          data={query.data}
          isadmin={isadmin}
        ></FolderNavigation>
      ) : (
        <DataTable
          storageKey="filesystem"
          query={query}
          columns={columns}
          toolbarConfig={toolbarConfig}
        >
          <TooltipButton
            variant="outline"
            size="icon"
            onClick={() => {
              handleReturnNavigation(backpath);
            }}
            className="h-8 w-8"
            disabled={isRoot}
            tooltipContent="返回上一级"
          >
            <ArrowLeftIcon className="size-4" />
          </TooltipButton>
          <FileUpload uploadPath={path} disabled={isRoot}></FileUpload>
          <Dialog>
            <DialogTrigger asChild>
              <TooltipButton
                variant="outline"
                className="h-8 w-8"
                size="icon"
                disabled={isRoot}
                tooltipContent="创建文件夹"
              >
                <FolderPlusIcon className="size-4" />
              </TooltipButton>
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
}
