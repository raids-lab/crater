import { useMemo, useState } from "react";
import { Input } from "@/components/ui/input";
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
import TooltipButton from "@/components/custom/TooltipButton";
import TooltipLink from "@/components/label/TooltipLink";
import { IUserAttributes } from "@/services/api/admin/user";
import PageTitle from "@/components/layout/PageTitle";
import TipBadge from "@/components/badge/TipBadge";
import { TimeDistance } from "@/components/custom/TimeDistance";
export interface DataItem {
  id: number;
  name: string;
  desc: string;
  username: string;
  createdAt?: string;
  tag: string[];
  url?: string;
  owner: IUserAttributes;
}

export default function DataList({
  items,
  title,
  actionArea,
}: {
  items: DataItem[];
  title: string;
  actionArea?: React.ReactNode;
}) {
  const [sort, setSort] = useState("ascending");
  const [modelType, setModelType] = useState("所有标签");
  const [searchTerm, setSearchTerm] = useState("");

  const tags = useMemo(() => {
    const tags = new Set<string>();
    items.forEach((model) => {
      model.tag.forEach((tag) => tags.add(tag));
    });
    return Array.from(tags);
  }, [items]);

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
        {filteredItems.map((item) => (
          <li
            key={item.name}
            className="bg-card flex flex-col justify-between gap-3 rounded-lg border hover:shadow-md"
          >
            <div className="flex flex-row items-center justify-between p-4 pb-0">
              <div className="flex items-center gap-2">
                <div
                  className={`bg-primary/10 text-primary flex size-10 items-center justify-center rounded-lg p-1`}
                >
                  {title === "模型" ? <BotIcon /> : <DatabaseZapIcon />}
                </div>
                <TooltipLink
                  to={`${item.id}`}
                  name={<p>{item.name}</p>}
                  tooltip={`查看${title}详情`}
                  className="font-semibold"
                />
              </div>
              {item.url && (
                <TooltipButton
                  tooltipContent={`更多操作`}
                  variant="ghost"
                  size="icon"
                >
                  <EllipsisVerticalIcon />
                </TooltipButton>
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
            <p className="text-muted-foreground line-clamp-2 px-4 text-sm text-balance">
              {item.desc}
            </p>
            <div>
              <div className="flex flex-row flex-wrap gap-1 p-4 pt-0">
                <TipBadge
                  title={item.owner?.name || item.username}
                  className="bg-purple-600/15 text-purple-600 hover:bg-purple-600/25"
                />
                <TipBadge
                  title={<TimeDistance date={item.createdAt || "2023"} />}
                />
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
