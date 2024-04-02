import type { FC } from "react";
import { useNavigate, useLocation, Link } from "react-router-dom";
import { createClient } from "webdav";
import { useState, useEffect, useMemo, useRef, Fragment } from "react";
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
  FileIcon,
  ArchiveIcon,
  HomeIcon,
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
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  //BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { logger } from "@/utils/loglevel";
//import { saveAs } from "file-saver";
//import { cn } from "@/lib/utils";

type FileStat = {
  filename: string;
  basename: string;
  lastmod: string;
  size: number;
  type: "file" | "directory";
  etag: string | null;
  mime?: string;
  //props?: DAVResultResponseProps;
};

export const Component: FC = () => {
  const location = useLocation();
  const path = location.pathname.replace("/portal/data/share", "");
  //const little_path = location.pathname.replace("/portal/data/share", "/share");
  const parts = path.split("/");
  const generatedArray = parts.map((part, index) => {
    const title = index === 0 ? "home" : part;
    const path1 = `/portal/data/share${parts.slice(0, index + 1).join("/")}`;

    return { title, path1 };
  });
  const backpath = `/portal/data/share${parts.slice(0, parts.length - 1).join("/")}`;
  const navigate = useNavigate();
  const client = createClient("https://crater.act.buaa.edu.cn/dufs/share", {
    username: "",
    password: "",
  });
  const [data, setData] = useState<FileStat[]>([]);
  const [dirName, setDirName] = useState<string>("");
  const contents = async () => await client.getDirectoryContents(`${path}`);

  const {
    data: rootList,
    isLoading,
    //dataUpdatedAt,
  } = useQuery({
    queryKey: ["root", "list"],
    queryFn: contents,
    select: (res) => res,
  });

  useEffect(() => {
    if (isLoading) return;
    if (!rootList) return;
    //async () => await client.createDirectory("/dnn-train-data/ceshi");
    const tableData: FileStat[] = (rootList as Array<FileStat>)
      .map((r) => {
        return {
          filename: r.filename,
          basename: r.basename,
          lastmod: r.lastmod,
          size: r.size,
          type: r.type,
          etag: r.etag,
          mime: r.mime,
        };
      })
      .sort((a, b) => {
        if (a.type === "directory" && b.type === "file") {
          return -1; // a在b之前
        } else if (a.type === "file" && b.type === "directory") {
          return 1; // a在b之后
        } else {
          return a.basename.localeCompare(b.basename);
        }
      });
    setData(tableData);
  }, [rootList, isLoading]);

  const columns = useMemo<ColumnDef<FileStat>[]>(
    () => [
      {
        id: "type",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"类型"} />
        ),
        cell: ({ row }) => {
          if (row.original.type == "directory") {
            return (
              <ArchiveIcon
                style={{
                  transform: "scale(1.4)",
                  color: "rgb(204, 153, 0)",
                }}
              />
              // </div>
            );
          } else {
            return (
              <FileIcon
                style={{
                  transform: "scale(1.4)",
                }}
              />
            );
          }
        },
        enableSorting: false,
        enableHiding: false,
      },

      {
        accessorKey: "basename",
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={getHeader("basename")}
          />
        ),
        cell: ({ row }) => {
          if (row.original.type == "directory") {
            return (
              <Button
                onClick={() =>
                  navigate(`/portal/data/share` + row.original.filename)
                }
                variant={"link"}
                className="h-8 px-0 text-left font-normal text-secondary-foreground"
              >
                {row.getValue("basename")}
              </Button>
            );
          } else {
            return <>{row.getValue("basename")}</>;
          }
        },
        enableSorting: false,
      },
      {
        accessorKey: "lastmod",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("lastmod")} />
        ),
        cell: ({ row }) => {
          return <TableDate date={row.getValue("lastmod")}></TableDate>;
        },
        //sortingFn: "datetime",
        enableSorting: false,
      },
      {
        accessorKey: "size",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("size")} />
        ),
        cell: ({ row }) => <FileSizeComponent size={row.getValue("size")} />,
      },
      {
        id: "actions",
        enableHiding: false,
        cell: ({ row }) => {
          if (row.original.type == "file") {
            return (
              <div className="flex flex-row space-x-1">
                <Button
                  variant="outline"
                  className="h-8 w-8 p-0 hover:text-sky-700"
                  title="下载文件"
                  onClick={() => {
                    const link = client.getFileDownloadLink(
                      `${row.original.filename}`,
                    );
                    const o = new XMLHttpRequest();
                    o.open("GET", link);
                    o.responseType = "blob";
                    o.onload = function () {
                      const content = o.response as string;
                      const a = document.createElement("a");
                      a.style.display = "none";
                      a.download = row.original.basename || "";
                      const blob = new Blob([content]);
                      a.href = URL.createObjectURL(blob);
                      document.body.appendChild(a);
                      a.click();
                      document.body.removeChild(a);
                    };
                    o.send();
                    toast.info("下载该文件");
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
    [client, navigate],
  );

  const getHeader = (key: string): string => {
    switch (key) {
      case "basename":
        return "文件名";
      case "lastmod":
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
      key: "basename",
    },
    filterOptions: [],
    getHeader: getHeader,
  };

  const refInput = useRef<HTMLInputElement>(null);
  const refInput2 = useRef<HTMLInputElement>(null);

  const { mutate: upload } = useMutation({
    mutationFn: (Files: File[]) => uploadFile(Files),
    onSuccess: () => {
      //await refetchRootList();
      toast.success("文件已上传");
    },
  });
  const handleFileSelect = ({
    currentTarget: { files },
  }: React.ChangeEvent<HTMLInputElement>) => {
    if (files && files.length) {
      const Files = Array.from(files);
      upload(Files);
      //上传文件2s后更新目录
      setTimeout(() => {
        queryClient
          .invalidateQueries({ queryKey: ["root", "list"] })
          .catch((err) => {
            logger.debug(err);
          });
      }, 1000);

      // uploadFile(Files).catch((err) => {
      //   logger.debug(err);
      // });
      // refetchRootList().catch((err) => {
      //   logger.debug(err);
      // });
      // toast.success("文件已上传");
    }
  };

  const uploadFile = async (Files: File[]) => {
    for (const file of Files) {
      if (file.name === undefined) return null;
      const filename = file.name.split("/").pop();
      if (filename === undefined) return null;
      const filedataBuffer = await file.arrayBuffer();
      await client.putFileContents(`${path}/` + filename, filedataBuffer, {
        overwrite: false,
      });
    }
    return null;
  };
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.persist();
    setDirName(e.target.value);
  };
  const queryClient = useQueryClient();
  const refetchRootList = async () => {
    try {
      await queryClient.invalidateQueries({ queryKey: ["root", "list"] });
    } catch (error) {
      logger.debug("更新目录失败", error);
    }
  };
  const clientCreateDir = async () => {
    //console.log("dirName:" + dirName);
    if (dirName != "") {
      await client.createDirectory(`${path}/` + dirName);
    }
  };
  const { mutate: CreateDir } = useMutation({
    mutationFn: () => clientCreateDir(),
    onSuccess: async () => {
      await refetchRootList();
      toast.success("文件夹已创建");
    },
  });
  return (
    <DataTable
      data={data}
      columns={columns}
      toolbarConfig={toolbarConfig}
      loading={isLoading}
    >
      {/* <div
            className="flex items-center space-x-1 md:space-x-2"
            aria-label="Breadcrumb"
          > */}
      <Breadcrumb>
        <BreadcrumbList>
          {generatedArray.map((item, index) => {
            return (
              <Fragment key={`bread-${index}`}>
                {/* {index !== 0 && (
                      <BreadcrumbSeparator key={`bread-separator-${index}`} />
                    )} */}
                {index !== 0 && "/"}
                {index === 0 && (
                  <BreadcrumbItem key={`bread-item-${index}`}>
                    {item.path1 && (
                      <BreadcrumbLink asChild>
                        <Link to={item.path1}>
                          <div
                            style={{
                              color: "rgb(65, 105, 225)",
                              transform: "scale(1.3)",
                              //backgroundColor: "red",
                            }}
                          >
                            <span style={{ color: "red" }}>
                              <HomeIcon
                                style={{
                                  color: "black",
                                  transform: "scale(1.15)",
                                  //fill: "red",
                                  //backgroundColor: "red",
                                }}
                              ></HomeIcon>
                            </span>
                          </div>
                        </Link>
                      </BreadcrumbLink>
                    )}
                    {!item.path1 && (
                      <BreadcrumbPage>{item.title}</BreadcrumbPage>
                    )}
                  </BreadcrumbItem>
                )}
                {index !== 0 && (
                  <BreadcrumbItem key={`bread-item-${index}`}>
                    {item.path1 && (
                      <BreadcrumbLink asChild>
                        <Link to={item.path1}>{item.title}</Link>
                      </BreadcrumbLink>
                    )}
                    {!item.path1 && (
                      <BreadcrumbPage>{item.title}</BreadcrumbPage>
                    )}
                  </BreadcrumbItem>
                )}
              </Fragment>
            );
          })}
        </BreadcrumbList>
      </Breadcrumb>

      {/* </div> */}
      <div style={{ marginLeft: "70px" }}></div>
      <Button
        className="h-8 w-8 p-0 hover:text-sky-700"
        onClick={() => navigate(backpath)}
      >
        <ArrowLeftIcon></ArrowLeftIcon>
      </Button>
      <Button
        onClick={() => {
          refInput.current?.click();
        }}
        variant="outline"
        className="h-8 w-8 p-0 hover:text-sky-700"
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
          <Button variant="outline" className="h-8 w-8 p-0 hover:text-sky-700">
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
