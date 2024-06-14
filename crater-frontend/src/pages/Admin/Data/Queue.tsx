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
import { TableDate } from "@/components/custom/TableDate";
import { FileSizeComponent } from "@/components/custom/FileSize";
import {
  DownloadIcon,
  UploadIcon,
  PlusIcon,
  ArrowLeftIcon,
} from "@radix-ui/react-icons";
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
import { File, Folder } from "lucide-react";
import { useSetAtom } from "jotai";
import { globalBreadCrumb } from "@/utils/store";
import {
  FileItem,
  apiGetQueueFiles,
  apiMkdir,
  apiUploadFile,
} from "@/services/api/file";
import { ACCESS_TOKEN_KEY } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";

export const Component: FC = () => {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [dirName, setDirName] = useState<string>("");
  const setBreadcrumb = useSetAtom(globalBreadCrumb);
  const path = useMemo(
    () => pathname.replace(/^\/admin\/data\/queue/, ""),
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
        };
      } else if (index == 2 && value == "queue") {
        return {
          title: "队列文件",
          path: "/admin/data/queue",
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
    queryKey: ["data", "queuefiles", path],
    queryFn: () => apiGetQueueFiles(`${path}`),
    select: (res) => res.data.data ?? [],
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
                <Folder className="mr-2 h-5 w-5 text-yellow-600 dark:text-yellow-400" />
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
                <File className="mr-2 h-5 w-5 text-muted-foreground" />
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
          return <TableDate date={row.getValue("modifytime")}></TableDate>;
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
                    const link = `https://crater.act.buaa.edu.cn/api/ss/download${path}/${row.original.name}`;
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
                  <DownloadIcon />
                </Button>
              </div>
            );
          }
        },
      },
    ],
    [navigate, path, pathname],
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

  const refInput = useRef<HTMLInputElement>(null);
  const refInput2 = useRef<HTMLInputElement>(null);

  const { mutate: upload } = useMutation({
    mutationFn: (Files: File[]) => uploadFile(Files),
    onSuccess: () => {
      toast.success("文件已上传");
      void queryClient.invalidateQueries({
        queryKey: ["data", "queuefiles", path],
      });
    },
  });

  const handleFileSelect = ({
    currentTarget: { files },
  }: React.ChangeEvent<HTMLInputElement>) => {
    if (files && files.length) {
      const Files = Array.from(files);
      upload(Files);
    }
  };

  const uploadFile = async (Files: File[]) => {
    for (const file of Files) {
      if (file.name === undefined) return null;
      const filename = file.name.split("/").pop();
      if (filename === undefined) return null;
      const filedataBuffer = await file.arrayBuffer();
      await apiUploadFile(`${path}/` + filename, filedataBuffer)
        .then(() => {
          toast.success("文件已下载");
        })
        .catch((error) => {
          showErrorToast(error);
        });
    }
    return null;
  };
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
        queryKey: ["data", "queuefiles", path],
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
        disabled={pathname === "/admin/data/queue"}
      >
        <ArrowLeftIcon />
      </Button>
      <Button
        onClick={() => {
          refInput.current?.click();
        }}
        size="icon"
        variant="outline"
        className="h-8 w-8 p-0"
        disabled={pathname === "/admin/data/queue"}
      >
        <UploadIcon />
      </Button>
      <input
        type="file"
        ref={refInput}
        style={{ display: "none" }}
        multiple={true}
        onChange={handleFileSelect}
      />
      <Dialog>
        <DialogTrigger asChild>
          <Button
            variant="outline"
            className="h-8 w-8"
            size="icon"
            disabled={pathname === "/admin/data/queue"}
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
