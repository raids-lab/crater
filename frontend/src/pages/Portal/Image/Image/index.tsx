import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useEffect, useState } from "react";
import { ColumnDef } from "@tanstack/react-table";
import { DataTableColumnHeader } from "@/components/custom/DataTable/DataTableColumnHeader";
import { DataTable } from "@/components/custom/DataTable";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import { ImageUploadForm } from "./UploadForm";
import { TimeDistance } from "@/components/custom/TimeDistance";
import {
  getHeader,
  imagepackPublicPersonalStatus,
  apiUserListImage,
  apiUserChangeImagePublicStatus,
  ImageInfoResponse,
  apiUserChangeImageDescription,
  apiUserCheckImageValid,
  ImageLinkPair,
  apiUserDeleteImageList,
  apiUserChangeImageTaskType,
} from "@/services/api/imagepack";
import { logger } from "@/utils/loglevel";
import { toast } from "sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui-custom/alert-dialog";
import {
  AlertCircle,
  BookCopy,
  Check,
  CheckCheck,
  CheckCircle,
  Globe,
  Lock,
  ListCheck,
  Loader2,
  Pencil,
  Tag,
  Trash2,
  Trash2Icon,
  X,
  AlertTriangle,
  SquareCheckBig,
} from "lucide-react";
import JobTypeLabel from "@/components/badge/JobTypeBadge";
import { useAtomValue } from "jotai";
import { globalUserInfo } from "@/utils/store";
import ImageLabel from "@/components/label/ImageLabel";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuItem,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from "@/components/ui/dropdown-menu";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { Input } from "@/components/ui/input";
import { JobType } from "@/services/api/vcjob";
import { Label } from "@radix-ui/react-label";

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: "搜索镜像",
    key: "imageLink",
  },
  filterOptions: [],
  getHeader: getHeader,
};

enum Dialogs {
  delete = "delete",
  status = "status",
  rename = "rename",
  valid = "valid",
}

interface ActionsProps {
  imageInfo: ImageInfoResponse;
  onChangeStatus: (id: number) => void;
  onUpdateDescription: (data: { id: number; description: string }) => void;
  linkPairs: ImageLinkPair[];
  onDeleteImageList: (ids: number[]) => void;
  userName: string;
  onChangeType: (data: { id: number; taskType: JobType }) => void;
}

const Actions: FC<ActionsProps> = ({
  imageInfo,
  onChangeStatus,
  onUpdateDescription,
  linkPairs,
  onDeleteImageList,
  userName,
  onChangeType,
}) => {
  const [openDialog, setOpenDialog] = useState(false);
  const [dialog, setDialog] = useState<Dialogs | undefined>(undefined);
  const isDisabled = imageInfo.creatorName !== userName;
  return (
    <div className="flex flex-row space-x-1">
      <AlertDialog open={openDialog} onOpenChange={setOpenDialog}>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">操作</span>
              <DotsHorizontalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="text-xs text-muted-foreground">
              操作
            </DropdownMenuLabel>
            <DropdownMenuItem
              onClick={() => {
                navigator.clipboard
                  .writeText(imageInfo.imageLink)
                  .then(() => {
                    toast.success("镜像地址已复制到剪贴板");
                  })
                  .catch((err) => {
                    toast.error("复制失败");
                    logger.error("复制失败", err);
                  });
              }}
            >
              <BookCopy className="text-blue-600" />
              复制链接
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => {
                setDialog(Dialogs.valid);
                setOpenDialog(true);
              }}
            >
              <CheckCheck className="text-green-600" />
              检验有效性
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={isDisabled}
              onClick={() => {
                setDialog(Dialogs.status);
                setOpenDialog(true);
              }}
            >
              {imageInfo.isPublic === true ? (
                <Lock className="text-amber-600" />
              ) : (
                <Globe className="text-green-600" />
              )}
              {imageInfo.isPublic === true ? "设为私有" : "设为公共"}
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={isDisabled}
              onClick={() => {
                setDialog(Dialogs.rename);
                setOpenDialog(true);
              }}
            >
              <ListCheck className="text-orange-600" />
              重命名
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={isDisabled}
              onClick={() => {
                setDialog(Dialogs.delete);
                setOpenDialog(true);
              }}
            >
              <Trash2Icon className="text-red-600" />
              删除镜像
            </DropdownMenuItem>

            <DropdownMenuSub>
              <DropdownMenuSubTrigger
                disabled={isDisabled}
                className={`${isDisabled ? "cursor-not-allowed opacity-50 data-[disabled]:pointer-events-none" : ""} group`}
              >
                <Tag className="text-cyan-600" />
                更改类型
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuRadioGroup value={`${JobType}`}>
                  {Object.keys(JobType).map((key) => (
                    <DropdownMenuRadioItem
                      key={key}
                      value={key}
                      disabled={[
                        JobType.WebIDE,
                        JobType.DeepSpeed,
                        JobType.KubeRay,
                        JobType.OpenMPI,
                      ].includes(JobType[key as keyof typeof JobType])}
                      onClick={() =>
                        onChangeType({
                          id: imageInfo.ID,
                          taskType: JobType[key as keyof typeof JobType],
                        })
                      }
                    >
                      {key}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuContent>
        </DropdownMenu>
        <AlertDialogContent>
          {dialog === Dialogs.delete ? (
            <DeleteDialog
              imageLinks={[imageInfo.imageLink]}
              onDeleteImageList={() => onDeleteImageList([imageInfo.ID])}
            />
          ) : dialog === Dialogs.status ? (
            <StatusDialog
              imageLink={imageInfo.imageLink}
              isPublic={imageInfo.isPublic}
              onChange={() => onChangeStatus(imageInfo.ID)}
            />
          ) : dialog === Dialogs.rename ? (
            <RenameDialog
              imageDescription={imageInfo.description}
              onRename={(newDescription) =>
                onUpdateDescription({
                  id: imageInfo.ID,
                  description: newDescription,
                })
              }
            />
          ) : dialog === Dialogs.valid ? (
            <ValidDialog
              linkPairs={linkPairs}
              onDeleteLinks={(invalidPairs: ImageLinkPair[]) => {
                onDeleteImageList(
                  invalidPairs
                    .filter((pair) => pair.creator === userName)
                    .map((pair) => pair.id),
                );
              }}
            />
          ) : null}
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export const Component: FC = () => {
  const queryClient = useQueryClient();
  const [openSheet, setOpenSheet] = useState(false);
  const [openCheckDialog, setCheckOpenDialog] = useState(false);
  const [selectedLinkPairs, setSelectedLinkPairs] = useState<ImageLinkPair[]>(
    [],
  );

  const user = useAtomValue(globalUserInfo);
  const imageInfo = useQuery({
    queryKey: ["imagelink", "list"],
    queryFn: () => apiUserListImage(),
    select: (res) => res.data.data.imageList,
  });

  const refetchImagePackList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["imagelink", "list"] }),
      ]);
    } catch (error) {
      logger.error("更新查询失败", error);
    }
  };
  const { mutate: deleteUserImageList } = useMutation({
    mutationFn: (idList: number[]) => apiUserDeleteImageList(idList),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像已删除");
    },
  });
  const { mutate: changeImagePublicStatus } = useMutation({
    mutationFn: (id: number) => apiUserChangeImagePublicStatus(id),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像状态更新");
    },
  });
  const { mutate: updateImageDescription } = useMutation({
    mutationFn: (data: { id: number; description: string }) =>
      apiUserChangeImageDescription(data),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像描述已更新");
    },
  });
  const { mutate: updateImageTaskType } = useMutation({
    mutationFn: (data: { id: number; taskType: JobType }) =>
      apiUserChangeImageTaskType(data),
    onSuccess: async () => {
      await refetchImagePackList();
      toast.success("镜像描述已更新");
    },
  });
  const columns: ColumnDef<ImageInfoResponse>[] = [
    {
      accessorKey: "taskType",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("taskType")} />
      ),
      cell: ({ row }) => {
        return <JobTypeLabel jobType={row.getValue("taskType")} />;
      },
    },
    {
      accessorKey: "imageLink",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("imageLink")} />
      ),
      cell: ({ row }) => (
        <ImageLabel
          description={row.original.description}
          url={row.getValue<string>("imageLink")}
        />
      ),
    },
    {
      accessorKey: "creatorName",
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={getHeader("creatorName")}
        />
      ),
      cell: ({ row }) => <div>{row.getValue("creatorName")}</div>,
    },
    {
      accessorKey: "isPublic",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("isPublic")} />
      ),
      cell: ({ row }) => {
        const imagePublicPersonalStatus = imagepackPublicPersonalStatus.find(
          (imagePublicPersonalStatus) =>
            imagePublicPersonalStatus.value === row.getValue("isPublic"),
        );
        return imagePublicPersonalStatus?.label;
      },
    },
    {
      accessorKey: "createdAt",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("createdAt")} />
      ),
      cell: ({ row }) => {
        return <TimeDistance date={row.getValue("createdAt")}></TimeDistance>;
      },
      sortingFn: "datetime",
    },
    {
      id: "actions",
      enableHiding: false,
      cell: ({ row }) => {
        const imageInfo = row.original;
        return (
          <Actions
            imageInfo={imageInfo}
            onChangeStatus={changeImagePublicStatus}
            onUpdateDescription={updateImageDescription}
            linkPairs={[
              {
                id: imageInfo.ID,
                imageLink: imageInfo.imageLink,
                description: imageInfo.description,
                creator: imageInfo.creatorName,
              },
            ]}
            onDeleteImageList={deleteUserImageList}
            userName={user.name}
            onChangeType={updateImageTaskType}
          />
        );
      },
    },
  ];

  return (
    <DataTable
      info={{
        title: "镜像列表",
        description: "展示可用的公共或私有镜像，在作业提交时可供选择",
      }}
      query={imageInfo}
      columns={columns}
      toolbarConfig={toolbarConfig}
      className="col-span-3"
      multipleHandlers={[
        {
          title: (rows) => `删除 ${rows.length} 个镜像链接`,
          description: (rows) => (
            <div className="rounded-md border border-destructive/20 bg-destructive/5 px-4 py-3">
              <div className="flex items-start gap-3">
                <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-destructive" />
                <div>
                  <p className="font-medium text-destructive">
                    以下镜像链接将被删除，确认要继续吗？
                  </p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {"『" +
                      rows
                        .map((row) => row.original.description)
                        .join("』,『") +
                      "』"}
                  </p>
                </div>
              </div>
            </div>
          ),
          icon: <Trash2Icon className="text-destructive" />,
          handleSubmit: (rows) => {
            const ids = rows.map((row) => row.original.ID);
            deleteUserImageList(ids);
          },
          isDanger: true,
        },
        {
          title: (rows) => `检测 ${rows.length} 个镜像链接`,
          description: (rows) => (
            <div className="rounded-md border border-green-600/20 bg-green-600/5 px-4 py-3">
              <div className="flex items-start gap-3">
                <SquareCheckBig className="mt-0.5 h-5 w-5 flex-shrink-0 text-green-600" />
                <div>
                  <p className="font-medium text-green-600">
                    以下镜像链接将被检测
                  </p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {"『" +
                      rows
                        .map((row) => row.original.description)
                        .join("』,『") +
                      "』"}
                  </p>
                </div>
              </div>
            </div>
          ),
          icon: <CheckCheck className="text-green-600" />,
          handleSubmit: (rows) => {
            setSelectedLinkPairs(
              rows.map((row) => ({
                id: row.original.ID,
                imageLink: row.original.imageLink,
                description: row.original.description,
                creator: row.original.creatorName,
              })),
            );
            setCheckOpenDialog(true);
          },
          isDanger: false,
        },
      ]}
    >
      <AlertDialog open={openCheckDialog} onOpenChange={setCheckOpenDialog}>
        <AlertDialogContent>
          <ValidDialog
            linkPairs={selectedLinkPairs}
            onDeleteLinks={(invalidPairs: ImageLinkPair[]) => {
              deleteUserImageList(
                invalidPairs
                  .filter((pair) => pair.creator === user.name)
                  .map((pair) => pair.id),
              );
            }}
          />
        </AlertDialogContent>
      </AlertDialog>

      <Sheet open={openSheet} onOpenChange={setOpenSheet}>
        <SheetTrigger asChild>
          <Button className="h-8 min-w-fit">导入镜像</Button>
        </SheetTrigger>
        <SheetContent className="max-h-screen overflow-y-auto sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>导入镜像链接</SheetTitle>
            <SheetDescription>导入一个新的训练作业镜像</SheetDescription>
          </SheetHeader>
          <Separator className="mt-4" />
          <ImageUploadForm closeSheet={() => setOpenSheet(false)} />
        </SheetContent>
      </Sheet>
    </DataTable>
  );
};

interface DeleteDialogProps {
  imageLinks: string[];
  onDeleteImageList: () => void;
}

export const DeleteDialog: FC<DeleteDialogProps> = ({
  imageLinks,
  onDeleteImageList,
}) => {
  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2 text-xl">
          <Trash2 className="h-5 w-5 text-destructive" />
          <span>删除镜像</span>
        </AlertDialogTitle>
      </AlertDialogHeader>

      <Separator className="my-3" />

      <AlertDialogDescription className="space-y-4 pt-2">
        <div className="rounded-md border border-destructive/20 bg-destructive/5 px-4 py-3">
          <div className="flex items-start gap-3">
            <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-destructive" />
            <div>
              <p className="font-medium text-destructive">将删除以下镜像</p>
              <p className="mt-1 text-sm text-muted-foreground">
                {"『" + imageLinks.join("』,『") + "』"}
              </p>
            </div>
          </div>
        </div>
      </AlertDialogDescription>

      <AlertDialogFooter>
        <AlertDialogCancel>
          <X />
          取消
        </AlertDialogCancel>

        <AlertDialogAction variant="destructive" onClick={onDeleteImageList}>
          <Trash2 />
          确认删除
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
};

interface StatusDialogProps {
  imageLink: string;
  isPublic: boolean;
  onChange: () => void;
}

export const StatusDialog: FC<StatusDialogProps> = ({
  imageLink,
  isPublic,
  onChange,
}) => {
  const currentStatus = isPublic ? "公共" : "私有";
  const newStatus = isPublic ? "私有" : "公共";

  // Choose icon based on the new status
  const StatusIcon = isPublic ? Lock : Globe;
  const statusColor = isPublic ? "text-amber-600" : "text-green-600";
  const bgColor = isPublic ? "bg-amber-50" : "bg-green-50";
  const darkBgColor = isPublic
    ? "dark:bg-amber-950/30"
    : "dark:bg-green-950/30";
  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-2 text-xl">
          <StatusIcon className={`h-5 w-5 ${statusColor}`} />
          <span>更新镜像访问权限</span>
        </AlertDialogTitle>
      </AlertDialogHeader>

      <Separator className="my-3" />

      <AlertDialogDescription className="space-y-4 pt-2">
        <div className="rounded-md bg-muted/50 px-4 py-3">
          <p className="text-sm text-muted-foreground">镜像链接</p>
          <p className="mt-1 break-all font-medium">『{imageLink}』</p>
        </div>

        <div className={`rounded-md ${bgColor} ${darkBgColor} p-4`}>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium">状态变更</p>
              <div className="flex items-center gap-6">
                <div className="flex items-center gap-1.5">
                  <div
                    className={`rounded-full ${isPublic ? "bg-green-100" : "bg-amber-100"} p-1`}
                  >
                    <div
                      className={`h-2 w-2 rounded-full ${isPublic ? "bg-green-500" : "bg-amber-500"}`}
                    ></div>
                  </div>
                  <span className="text-sm">{currentStatus}</span>
                </div>
                <div className="text-muted-foreground">→</div>
                <div className="flex items-center gap-1.5">
                  <div
                    className={`rounded-full ${!isPublic ? "bg-green-100" : "bg-amber-100"} p-1`}
                  >
                    <div
                      className={`h-2 w-2 rounded-full ${!isPublic ? "bg-green-500" : "bg-amber-500"}`}
                    ></div>
                  </div>
                  <span className="text-sm font-medium">{newStatus}</span>
                </div>
              </div>
            </div>
            <StatusIcon className={`h-8 w-8 ${statusColor} opacity-20`} />
          </div>
        </div>

        <p className="text-sm text-muted-foreground">
          {isPublic
            ? "设为私有后，只有您可以访问此镜像。"
            : "设为公共后，任何人都可以通过链接访问此镜像。"}
        </p>
      </AlertDialogDescription>

      <AlertDialogFooter>
        <AlertDialogCancel>
          <X />
          取消
        </AlertDialogCancel>

        <AlertDialogAction
          className={`flex items-center gap-2 ${isPublic ? "bg-amber-600 hover:bg-amber-700" : "bg-green-600 hover:bg-green-700"}`}
          onClick={onChange}
        >
          <Check />
          确认
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
};

interface RenameDialogProps {
  imageDescription: string;
  onRename: (newDescription: string) => void;
}

export const RenameDialog: FC<RenameDialogProps> = ({
  imageDescription,
  onRename,
}) => {
  const initialDescription = "";
  const [newDescription, setNewDescription] = useState(initialDescription);
  const [isTouched, setIsTouched] = useState(false);
  useEffect(() => {
    setNewDescription(initialDescription || "");
    setIsTouched(false);
  }, [initialDescription]);

  const isValid = newDescription.trim().length > 0;
  const hasChanged = newDescription !== initialDescription;
  const canSubmit = isValid && hasChanged;
  return (
    <>
      <AlertDialogHeader>
        <AlertDialogTitle className="flex items-center gap-1 text-xl">
          <Pencil className="h-5 w-5 text-primary" />
          <span>更新镜像名称</span>
        </AlertDialogTitle>
      </AlertDialogHeader>

      <Separator className="my-3" />

      <AlertDialogDescription className="space-y-4 pt-2">
        <div className="rounded-md bg-muted/50 px-4 py-3">
          <p className="text-sm text-muted-foreground">当前名称</p>
          <p className="mt-1 font-medium">「{imageDescription}」</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="new-description" className="font-medium">
            新名称
          </Label>
          <Input
            id="new-description"
            type="text"
            value={newDescription}
            onChange={(e) => {
              setNewDescription(e.target.value);
              setIsTouched(true);
            }}
            placeholder="输入新的描述"
            className={`transition-all ${isTouched && !isValid ? "border-destructive ring-destructive/10" : ""}`}
            autoFocus
          />
          {isTouched && !isValid && (
            <p className="text-xs text-destructive">请输入有效的名称</p>
          )}
        </div>
      </AlertDialogDescription>

      <AlertDialogFooter className="mt-4 gap-2 sm:gap-0">
        <AlertDialogCancel asChild>
          <Button variant="outline" className="flex items-center gap-2">
            <X className="h-4 w-4" />
            取消
          </Button>
        </AlertDialogCancel>

        <AlertDialogAction asChild>
          <Button
            variant="default"
            className="flex items-center gap-2"
            onClick={() => onRename(newDescription)}
            disabled={!canSubmit}
          >
            <Check className="h-4 w-4" />
            确认
          </Button>
        </AlertDialogAction>
      </AlertDialogFooter>
    </>
  );
};

interface ValidDialogProps {
  linkPairs: ImageLinkPair[];
  onDeleteLinks: (invalidPairs: ImageLinkPair[]) => void;
}

export const ValidDialog: FC<ValidDialogProps> = ({
  linkPairs,
  onDeleteLinks,
}) => {
  const { data, isLoading } = useQuery({
    queryKey: ["checkImageValid", linkPairs],
    queryFn: () => apiUserCheckImageValid({ linkPairs }),
    select: (res) => res.data.data.linkPairs,
  });

  const invalidPairs = data;
  const isValid = !isLoading && invalidPairs?.length === 0;

  return (
    <>
      {isLoading ? (
        <div className="flex flex-col items-center justify-center bg-background/80 backdrop-blur-sm">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
          <span className="mt-2 text-sm font-medium">验证镜像中...</span>
        </div>
      ) : (
        <>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2 text-xl">
              {isValid ? (
                <>
                  <CheckCircle className="h-5 w-5 text-green-500" />
                  <span>所选镜像有效</span>
                </>
              ) : (
                <>
                  <AlertCircle className="h-5 w-5 text-destructive" />
                  <span>无效镜像链接</span>
                </>
              )}
            </AlertDialogTitle>
          </AlertDialogHeader>

          <Separator className="my-3" />

          <AlertDialogDescription className="pt-2">
            {isValid ? (
              <div className="flex h-16 items-center justify-center rounded-md bg-green-50 px-4 py-3 dark:bg-green-950/30">
                <span className="flex items-center gap-2 text-center text-sm text-green-700 dark:text-green-400">
                  <CheckCheck className="h-4 w-4" />
                  所选镜像链接有效，可以继续操作。
                </span>
              </div>
            ) : (
              <div className="max-h-[240px] space-y-3 overflow-y-auto pr-1">
                {invalidPairs?.map((pair) => (
                  <div className="rounded-md border border-destructive/20 bg-destructive/5 px-4 py-3 transition-all hover:bg-destructive/10">
                    <div className="space-y-1 text-sm">
                      <div className="group relative overflow-hidden text-ellipsis text-muted-foreground">
                        <span className="font-medium text-foreground">
                          链接:{" "}
                        </span>
                        <span className="break-all">{pair.imageLink}</span>
                      </div>
                      <div className="text-muted-foreground">
                        <span className="font-medium text-foreground">
                          描述:{" "}
                        </span>
                        {pair.description || (
                          <span className="italic text-muted-foreground">
                            无描述
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </AlertDialogDescription>

          <AlertDialogFooter className="mt-4 gap-2 sm:gap-0">
            <AlertDialogCancel className="flex items-center gap-2">
              <X className="h-4 w-4" />
              关闭
            </AlertDialogCancel>

            {(invalidPairs?.length ?? 0) > 0 && (
              <AlertDialogAction
                variant="destructive"
                className="flex items-center gap-2"
                onClick={() => onDeleteLinks(invalidPairs ?? [])}
              >
                <Trash2 className="h-4 w-4" />
                删除链接
              </AlertDialogAction>
            )}
          </AlertDialogFooter>
        </>
      )}
    </>
  );
};
