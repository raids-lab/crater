import type { FC } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { createClient } from "webdav";
import { useState, useEffect, useMemo, useRef } from "react";
import { DataTable } from "@/components/custom/OldDataTable";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { DataTableColumnHeader } from "@/components/custom/OldDataTable/DataTableColumnHeader";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableToolbarConfig } from "@/components/custom/OldDataTable/DataTableToolbar";
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
import { useSetRecoilState } from "recoil";
import { globalBreadCrumb } from "@/utils/store";
import {
  Fileresp,
  apiGetFiles,
  apiMkdir,
  apiUploadFile,
} from "@/services/api/file";
import { ACCESS_TOKEN_KEY } from "@/utils/store";
import { showErrorToast } from "@/utils/toast";

//import { saveAs } from "file-saver";
//import { cn } from "@/lib/utils";

// type FileStat = {
//   filename: string;
//   basename: string;
//   lastmod: string;
//   size: number;
//   type: "file" | "directory";
//   etag: string | null;
//   mime?: string;
//   //props?: DAVResultResponseProps;
// };

const client = createClient("https://crater.act.buaa.edu.cn/api/ss", {
  username: "",
  password: "",
});
client.getFileContents;
export const Component: FC = () => {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  // const [data, setData] = useState<Fileresp[]>([]);
  const [dirName, setDirName] = useState<string>("");
  const [filepath, setFilepath] = useState<string>("");
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  const path = useMemo(
    () => filepath.replace(/^\/portal\/data\/share/, ""),
    [filepath],
  );

  useEffect(() => {
    const pathParts = pathname.split("/").filter(Boolean);
    const breadcrumb = pathParts.map((value, index) => {
      if (index == 0 && value == "portal") {
        return {
          title: "Portal",
        };
      } else if (index == 1 && value == "data") {
        return {
          title: "数据管理",
          path: "/portal/data",
        };
      } else if (index == 2 && value == "share") {
        return {
          title: "项目文件",
          path: "/portal/data/share",
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

  const backfilepath = useMemo(() => {
    return filepath.replace(/\/[^/]+$/, "");
  }, [filepath]);

  const { data: rootList, isLoading } = useQuery({
    queryKey: ["data", "share", path],
    queryFn: () => apiGetFiles(`${path}`),
    select: (res) => res.data.data,
  });
  const data: Fileresp[] = useMemo(() => {
    if (!rootList) {
      return [];
    }
    return rootList
      .map((r) => {
        return {
          name: r.name,
          modifytime: r.modifytime,
          filename: r.filename,
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
      });
  }, [rootList]);

  // useEffect(() => {
  //   if (isLoading) return;
  //   if (!rootList) return;
  //   const tableData: Fileresp[] = rootList
  //     .map((r) => {
  //       return {
  //         name: r.name,
  //         modifytime: r.modifytime,
  //         isdir: r.isdir,
  //         size: r.size,
  //         sys: r.sys,
  //       };
  //     })
  //     .sort((a, b) => {
  //       if (a.isdir && !b.isdir) {
  //         return -1; // a在b之前
  //       } else if (!a.isdir && b.isdir) {
  //         return 1; // a在b之后
  //       } else {
  //         return a.name.localeCompare(b.name);
  //       }
  //     });
  //   setData(tableData);
  // }, [rootList, isLoading]);

  const columns = useMemo<ColumnDef<Fileresp>[]>(
    () => [
      {
        id: "type",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"类型"} />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return (
              <Folder className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
              // </div>
            );
          } else {
            return <File className="h-5 w-5 text-muted-foreground" />;
          }
        },
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: "filename",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("filename")}
          />
        ),
        cell: ({ row }) => {
          if (row.original.isdir) {
            return (
              <Button
                onClick={() => {
                  setFilepath(filepath + "/" + row.original.name);
                  navigate(pathname + "/" + row.original.filename);
                }}
                variant={"link"}
                className="h-8 px-0 text-left font-normal text-secondary-foreground"
              >
                {row.getValue("filename")}
              </Button>
            );
          } else {
            return <>{row.getValue("filename")}</>;
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
            return <FileSizeComponent size={0} />;
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
                    const link = client.getFileDownloadLink(
                      `download${path}/${row.original.name}`,
                    );
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
    [navigate, path, pathname, filepath],
  );

  const getHeader = (key: string): string => {
    switch (key) {
      case "filename":
        return "文件名";
      case "modifytime":
        return "更新于";
      case "size":
        return "文件大小";
      default:
        return key;
    }
  };
  const toolbarConfig: DataTableToolbarConfig<string> = {
    filterInput: {
      placeholder: "搜索名称",
      key: "filename",
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
      void queryClient.invalidateQueries({ queryKey: ["data", "share", path] });
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
      void queryClient.invalidateQueries({ queryKey: ["data", "share", path] });
    },
  });

  return (
    <DataTable
      data={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      loading={isLoading}
      className="col-span-3"
    >
      <Button
        variant="outline"
        size="icon"
        onClick={() => {
          setFilepath(backfilepath);
          navigate(backpath);
        }}
        className="h-8 w-8"
        disabled={pathname === "/portal/data/share"}
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
        disabled={pathname === "/portal/data/share"}
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
            disabled={pathname === "/portal/data/share"}
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
