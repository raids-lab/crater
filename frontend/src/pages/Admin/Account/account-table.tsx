import { DataTableColumnHeader } from "@/components/custom/PagenationDataTable/DataTableColumnHeader";
import { IAccount } from "@/services/api/account";
import { Button } from "@/components/ui/button";
import { PencilIcon, TrashIcon, UserRoundIcon } from "lucide-react";
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
import { ColumnDef } from "@tanstack/react-table";
import ResourceBadges from "@/components/label/ResourceBadges";
import { DataTableToolbarConfig } from "@/components/custom/DataTable/DataTableToolbar";
import { Link } from "react-router-dom";
import Identicon from "@polkadot/react-identicon";
import { stringToSS58 } from "@/utils/ss58";
import { format, formatDistanceToNow } from "date-fns";
import { zhCN } from "date-fns/locale";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const getHeader = (key: string): string => {
  switch (key) {
    case "nickname":
      return "账户";
    case "deserved":
      return "应得";
    case "guaranteed":
      return "保证";
    case "capability":
      return "上限";
    default:
      return key;
  }
};

export const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    key: "name",
    placeholder: "搜索账户名称",
  },
  filterOptions: [],
  getHeader: getHeader,
};

export const getColumns = (
  handleEdit: (project: IAccount) => void,
  handleDelete: (project: IAccount) => void,
): ColumnDef<IAccount>[] => {
  return [
    {
      accessorKey: "nickname",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("nickname")} />
      ),
      cell: ({ row }) => {
        const expiredAt = row.original.expiredAt
          ? new Date(row.original.expiredAt)
          : undefined;
        const diff = expiredAt ? expiredAt.getTime() - Date.now() : 0;
        return (
          <Link
            to={`${row.original.id}`}
            className="flex flex-row items-center justify-start gap-2 hover:text-primary"
          >
            <Identicon
              value={stringToSS58(row.original.name)}
              size={32}
              theme="substrate"
              className="!cursor-pointer"
            />
            <div className="flex flex-col items-start gap-0.5">
              {row.getValue("nickname")}
              {expiredAt && (
                <TooltipProvider delayDuration={10}>
                  <Tooltip>
                    <TooltipTrigger>
                      <span className="text-xs text-muted-foreground">
                        {diff < 0 ? (
                          <>已过期</>
                        ) : (
                          <>
                            {formatDistanceToNow(expiredAt, {
                              locale: zhCN,
                              addSuffix: true,
                            })}
                            有效
                          </>
                        )}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent
                      side="right"
                      className="border bg-background text-foreground"
                    >
                      {format(expiredAt, "PPP", { locale: zhCN })}过期
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </div>
          </Link>
        );
      },
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
        return <ResourceBadges resources={row.original.quota.guaranteed} />;
      },
      enableSorting: false,
    },
    {
      accessorKey: "deserved",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader("deserved")} />
      ),
      cell: ({ row }) => {
        return <ResourceBadges resources={row.original.quota.deserved} />;
      },
      enableSorting: false,
    },
    {
      accessorKey: "capability",
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={getHeader("capability")}
        />
      ),
      cell: ({ row }) => {
        return <ResourceBadges resources={row.original.quota.capability} />;
      },
      enableSorting: false,
    },
    {
      id: "actions",
      enableHiding: false,
      cell: ({ row }) => {
        return (
          <div className="flex flex-row items-center justify-center gap-1">
            <Link to={`${row.original.id}`}>
              <Button
                title="管理用户"
                variant="outline"
                className="h-8 w-8"
                size="icon"
              >
                <UserRoundIcon className="size-4" />
              </Button>
            </Link>
            <Button
              title="修改配额"
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => handleEdit(row.original)}
            >
              <PencilIcon className="size-4" />
            </Button>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  title="删除账户"
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                >
                  <TrashIcon className="size-4" />
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除账户</AlertDialogTitle>
                  <AlertDialogDescription>
                    账户 {row.original.nickname} 将被删除，请谨慎操作。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => handleDelete(row.original)}
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
  ];
};
