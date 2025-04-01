import { useMemo, useState } from "react";
import { Input } from "@/components/ui/input";
//import { Link } from "react-router-dom";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import {
  ArrowDownAZIcon,
  ArrowDownZAIcon,
  BotIcon,
  DatabaseZapIcon,
  EllipsisVerticalIcon,
  SearchIcon,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
//import TooltipButton from "@/components/custom/TooltipButton";
import TooltipLink from "@/components/label/TooltipLink";
import { IUserAttributes } from "@/services/api/admin/user";
import PageTitle from "@/components/layout/PageTitle";
import TipBadge from "@/components/badge/TipBadge";
import { TimeDistance } from "@/components/custom/TimeDistance";
import { motion } from "framer-motion";
import { globalUserInfo } from "@/utils/store";
import { useAtomValue } from "jotai";
// 导入所需组件
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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

import { Trash2Icon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

export interface DataItem {
  id: number;
  name: string;
  desc: string;
  username: string;
  createdAt?: string;
  tag: string[];
  url?: string;
  template?: string;
  owner: IUserAttributes;
}
//假设 JobType 是这样定义的枚举
enum JobType {
  Jupyter = "jupyter",
  Custom = "custom",
  Tensorflow = "tensorflow",
  Pytorch = "pytorch",
}
export default function DataList({
  items,
  title,
  actionArea,
  isWIP,
  itemdelete,
  onRefresh,
}: {
  items: DataItem[];
  title: string;
  actionArea?: React.ReactNode;
  isWIP?: boolean;
  itemdelete?: (id: number) => void;
  onRefresh?: () => void;
}) {
  const [sort, setSort] = useState("ascending");
  const [modelType, setModelType] = useState("所有标签");
  const [searchTerm, setSearchTerm] = useState("");
  const user = useAtomValue(globalUserInfo);
  const tags = useMemo(() => {
    const tags = new Set<string>();
    items.forEach((model) => {
      model.tag.forEach((tag) => tags.add(tag));
    });
    return Array.from(tags);
  }, [items]);
  const getNewJobUrl = (jobType: JobType) => {
    switch (jobType) {
      case JobType.Jupyter: // 直接匹配枚举值
        return "job/iner/new-jupyter-vcjobs";
      case JobType.Custom:
        return "job/batch/new-vcjobs";
      case JobType.Tensorflow:
        return "job/batch/new-tensorflow";
      case JobType.Pytorch:
        return "job/batch/new-pytorch";
      default:
        return "job/batch/new-vcjobs";
    }
  };
  // 新增 JSON 解析函数
  const getJobUrlFromTemplate = (template: string): string => {
    try {
      const parsed = JSON.parse(template);

      // 类型安全校验
      if (!parsed.type || !Object.values(JobType).includes(parsed.type)) {
        return getNewJobUrl(JobType.Jupyter);
      }

      // 通过类型断言确保类型安全
      const jobType = parsed.type as JobType;
      return getNewJobUrl(jobType);
    } catch {
      return getNewJobUrl(JobType.Jupyter); // 解析失败返回默认
    }
  };
  const filteredItems = items
    .sort((a, b) =>
      sort === "descending"
        ? new Date(a.createdAt || "").getTime() -
          new Date(b.createdAt || "").getTime()
        : new Date(b.createdAt || "").getTime() -
          new Date(a.createdAt || "").getTime(),
    )
    .filter((item) =>
      modelType === "所有标签"
        ? true
        : item.tag.includes(modelType)
          ? true
          : false,
    )
    .filter((item) =>
      item.name.toLowerCase().includes(searchTerm.toLowerCase()),
    );

  return (
    <div>
      <PageTitle
        title={title}
        isWIP={isWIP}
        description={`我们为您准备了一些常见${title}，也欢迎您上传并分享更多${title}。`}
      >
        {actionArea}
      </PageTitle>
      <div className="my-4 flex items-end justify-between sm:my-0 sm:items-center">
        <div className="flex flex-col gap-4 sm:my-4 sm:flex-row">
          <div className="relative ml-auto h-9 flex-1 md:grow-0">
            <SearchIcon className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
            <Input
              placeholder={`搜索${title}...`}
              className="h-9 w-40 pl-8 lg:w-[250px]"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          <Select value={modelType} onValueChange={setModelType}>
            <SelectTrigger className="min-w-36">
              <SelectValue>{modelType}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="所有标签">所有标签</SelectItem>
              {tags.map((tag) => (
                <SelectItem key={tag} value={tag}>
                  {tag}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Select value={sort} onValueChange={setSort}>
          <SelectTrigger className="w-16">
            <SelectValue>
              {sort === "ascending" ? (
                <ArrowDownAZIcon size={16} />
              ) : (
                <ArrowDownZAIcon size={16} />
              )}
            </SelectValue>
          </SelectTrigger>
          <SelectContent align="end">
            <SelectItem value="ascending">
              <div className="flex items-center gap-4">
                <ArrowDownAZIcon size={16} />
                <span>升序</span>
              </div>
            </SelectItem>
            <SelectItem value="descending">
              <div className="flex items-center gap-4">
                <ArrowDownZAIcon size={16} />
                <span>降序</span>
              </div>
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <Separator />
      <ul className="faded-bottom no-scrollbar grid gap-4 overflow-auto pt-4 pb-16 md:grid-cols-2 lg:grid-cols-3">
        {filteredItems.map((item, index) => (
          <motion.li
            key={item.name}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: (index / 3) * 0.1 }}
            whileHover={{ y: -5 }}
            className="bg-card flex flex-col justify-between gap-3 rounded-lg border hover:shadow-md"
          >
            <div className="flex flex-row items-center justify-between p-4 pb-0">
              <div className="flex items-center gap-2">
                <div
                  className={`bg-primary/10 text-primary flex size-10 items-center justify-center rounded-lg p-1`}
                >
                  {title === "模型" ? <BotIcon /> : <DatabaseZapIcon />}
                </div>
                {title === "作业模板" ? (
                  <TooltipLink
                    to={`/portal/${getJobUrlFromTemplate(item.template || "")}?fromTemplate=${item.id}`}
                    name={<p className="text-left">{item.name}</p>}
                    tooltip={`查看${title}详情`}
                    className="font-semibold"
                  />
                ) : (
                  <TooltipLink
                    to={`${item.id}`}
                    name={<p className="text-left">{item.name}</p>}
                    tooltip={`查看${title}详情`}
                    className="font-semibold"
                  />
                )}
              </div>
              {title === "作业模板" && (
                <AlertDialog>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 p-0"
                      >
                        <span className="sr-only">更多操作</span>
                        <EllipsisVerticalIcon className="size-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuLabel className="text-muted-foreground text-xs">
                        操作
                      </DropdownMenuLabel>
                      {user?.name === item.owner.name && (
                        <AlertDialogTrigger asChild>
                          <DropdownMenuItem className="group text-red-500">
                            <Trash2Icon className="text-destructive mr-2 size-4" />
                            删除
                          </DropdownMenuItem>
                        </AlertDialogTrigger>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>删除作业模板</AlertDialogTitle>
                      <AlertDialogDescription>
                        作业模板 {item.name} 将被删除，此操作不可恢复。
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>取消</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
                        onClick={async (e) => {
                          e.stopPropagation();
                          if (itemdelete && onRefresh) {
                            try {
                              await itemdelete(item.id);
                              // 成功删除后，重置搜索和过滤条件以刷新列表
                              setSearchTerm("");
                              setModelType("所有标签");
                              // 如果父组件提供了重新获取数据的方法，可以在这里调用
                              onRefresh();
                            } catch (error) {
                              toast.error("删除失败:" + error);
                            }
                          }
                        }}
                      >
                        删除
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              )}
            </div>

            {item.tag.length > 0 && (
              <div className="flex flex-row flex-wrap gap-1 px-4 pb-1">
                {item.tag.map((tag) => (
                  <Badge variant="secondary" key={tag} className="rounded-full">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}
            <p
              className="text-muted-foreground line-clamp-3 px-4 text-sm text-balance"
              title={item.desc}
            >
              {item.desc}
            </p>
            <div>
              <div className="flex flex-row flex-wrap gap-1 p-4 pt-0">
                <TipBadge
                  title={item.owner?.nickname || item.username}
                  className="bg-purple-600/15 text-purple-600 hover:bg-purple-600/25"
                />
                <TipBadge
                  title={<TimeDistance date={item.createdAt || "2023"} />}
                />
              </div>
            </div>
          </motion.li>
        ))}
      </ul>
    </div>
  );
}
