// i18n-processed-v1.1.0
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
import ResourceBadges from "@/components/badge/ResourceBadges";
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
import { TFunction } from "i18next";

const getHeader = (key: string, t: (key: string) => string): string => {
  switch (key) {
    case "nickname":
      return t("table.headers.nickname");
    case "deserved":
      return t("table.headers.deserved");
    case "guaranteed":
      return t("table.headers.guaranteed");
    case "capability":
      return t("table.headers.capability");
    default:
      return key;
  }
};

export const getToolbarConfig = (
  t: (key: string) => string,
): DataTableToolbarConfig => {
  return {
    filterInput: {
      key: "name",
      placeholder: "搜索账户名称",
    },
    filterOptions: [],
    getHeader: (key: string) => getHeader(key, t),
  };
};

export const getColumns = (
  handleEdit: (project: IAccount) => void,
  handleDelete: (project: IAccount) => void,
  t: TFunction<"translation", undefined>,
): ColumnDef<IAccount>[] => {
  return [
    {
      accessorKey: "nickname",
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={getHeader("nickname", t)}
        />
      ),
      cell: ({ row }) => {
        const expiredAt = row.original.expiredAt
          ? new Date(row.original.expiredAt)
          : undefined;
        const diff = expiredAt ? expiredAt.getTime() - Date.now() : 0;
        return (
          <Link
            to={`${row.original.id}`}
            className="hover:text-primary flex flex-row items-center justify-start gap-2"
          >
            <Identicon
              value={stringToSS58(row.original.name)}
              size={32}
              theme="substrate"
              className="cursor-pointer!"
            />
            <div className="flex flex-col items-start gap-0.5">
              {row.getValue("nickname")}
              {expiredAt && (
                <TooltipProvider delayDuration={10}>
                  <Tooltip>
                    <TooltipTrigger>
                      <span className="text-muted-foreground text-xs">
                        {diff < 0 ? (
                          t("table.tooltip.expired")
                        ) : (
                          <>
                            {formatDistanceToNow(expiredAt, {
                              locale: zhCN,
                              addSuffix: true,
                            })}
                            {t("table.tooltip.valid")}
                          </>
                        )}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent
                      side="right"
                      className="bg-background text-foreground border"
                    >
                      {format(expiredAt, "PPP", { locale: zhCN })}{" "}
                      {t("table.tooltip.expires")}
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
          title={getHeader("guaranteed", t)}
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
        <DataTableColumnHeader
          column={column}
          title={getHeader("deserved", t)}
        />
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
          title={getHeader("capability", t)}
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
                title={t("table.actions.manageUser")}
                variant="outline"
                className="h-8 w-8"
                size="icon"
              >
                <UserRoundIcon className="size-4" />
              </Button>
            </Link>
            <Button
              title={t("table.actions.editQuota")}
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
                  title={t("table.actions.delete")}
                  variant="outline"
                  size="icon"
                  className="h-8 w-8"
                >
                  <TrashIcon className="size-4" />
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {t("table.actions.deleteTitle")}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    {t("table.actions.deleteDescription", {
                      name: row.original.nickname,
                    })}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>
                    {t("table.actions.cancel")}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => handleDelete(row.original)}
                  >
                    {t("table.actions.delete")}
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
