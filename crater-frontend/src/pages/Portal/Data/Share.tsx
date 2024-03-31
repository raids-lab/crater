import type { FC } from "react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
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
import {
  DownloadIcon,
  UploadIcon,
  PlusIcon,
  FileIcon,
  ArchiveIcon,
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
import { logger } from "@/utils/loglevel";

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
    const tableData: FileStat[] = (rootList as Array<FileStat>).map((r) => {
      return {
        filename: r.filename,
        basename: r.basename,
        lastmod: r.lastmod,
        size: r.size,
        type: r.type,
        etag: r.etag,
        mime: r.mime,
      };
    });
    setData(tableData);
  }, [rootList, isLoading]);

  const columns = useMemo<ColumnDef<FileStat>[]>(
    () => [
      {
        id: "type",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={"type"} />
        ),
        cell: ({ row }) => {
          if (row.original.type == "directory") {
            return (
              <ArchiveIcon />
              // </div>
            );
          } else {
            return <FileIcon />;
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
        cell: ({ row }) => <>{row.getValue("lastmod")}</>,
      },
      {
        accessorKey: "size",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader("size")} />
        ),
        cell: ({ row }) => <>{row.getValue("size")}</>,
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
                      row.original.filename,
                    );
                    const a = document.createElement("a");
                    a.style.display = "none";
                    a.href = `/lib/attach/macth/download?fileUrl=${link}`;
                    a.download = row.original.basename || "";
                    document.body.appendChild(a);
                    a.click();
                    document.body.appendChild(a);
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
  const handleFileSelect = ({
    currentTarget: { files },
  }: React.ChangeEvent<HTMLInputElement>) => {
    if (files && files.length) {
      const Files = Array.from(files);
      uploadFile(Files).catch((err) => {
        logger.debug(err);
      });
      refetchRootList().catch((err) => {
        logger.debug(err);
      });
      toast.success("文件已上传");
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
    <Card className="h-[calc(100vh_-104px)] w-full">
      <CardHeader className="py-3"></CardHeader>
      <CardContent className="relative">
        <DataTable
          data={data}
          columns={columns}
          toolbarConfig={toolbarConfig}
          loading={isLoading}
        >
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
              <Button
                variant="outline"
                className="h-8 w-8 p-0 hover:text-sky-700"
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
      </CardContent>
    </Card>
  );
};
