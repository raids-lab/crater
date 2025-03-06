import { useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { LogInIcon, Folder, FolderOpen, ChevronRight } from "lucide-react";
import { getFolderTitle } from "@/components/file/LazyFileTree";
import { useLocation, useNavigate } from "react-router-dom";
import { FileItem } from "@/services/api/file";
import { cn } from "@/lib/utils";
import PageTitle from "@/components/layout/PageTitle";

// TODO: 修改硬编码
const isPublicFolder = (folder: string) => folder === "sugon-gpu-incoming";

const isAccountFolder = (folder: string) => folder.startsWith("q-");

const isUserFolder = (folder: string) =>
  !isPublicFolder(folder) && !isAccountFolder(folder);

const getFolderDescription = (folder: string) => {
  // public: 公共空间
  // q-*: 账户共享空间
  // u-*: 用户私有空间
  if (isPublicFolder(folder)) {
    return "平台上所有用户都可以访问，如需获取上传权限，请联系管理员。";
  } else if (isAccountFolder(folder)) {
    return "账户共享空间，仅限账户内用户访问，如需获取上传权限，请联系账户管理员。";
  }
  return "个人空间，仅限当前用户访问，拥有完全的读写权限。";
};

export default function FolderNavigation({
  data: rowData,
}: {
  data?: FileItem[];
}) {
  const [hoveredFolder, setHoveredFolder] = useState<string | null>(null);
  const { pathname } = useLocation();
  const navigate = useNavigate();

  // 对文件夹进行排序，公共 -> 账户 -> 用户
  const sortFolders = (folders: FileItem[]) => {
    return folders.sort((a, b) => {
      if (isPublicFolder(a.name)) {
        return -1;
      } else if (isAccountFolder(a.name) && isUserFolder(b.name)) {
        return -1;
      }
      return 1;
    });
  };

  const data = useMemo(() => sortFolders(rowData || []), [rowData]);

  // 为每个文件夹分配一个固定的颜色
  const getFolderColor = (index: number) => {
    const colors = [
      {
        bg: "bg-blue-50 dark:bg-blue-950/30",
        border: "border-blue-200 dark:border-blue-800",
        icon: "text-blue-600 dark:text-blue-400",
      },
      {
        bg: "bg-green-50 dark:bg-green-950/30",
        border: "border-green-200 dark:border-green-800",
        icon: "text-green-600 dark:text-green-400",
      },
      {
        bg: "bg-purple-50 dark:bg-purple-950/30",
        border: "border-purple-200 dark:border-purple-800",
        icon: "text-purple-600 dark:text-purple-400",
      },
    ];

    return colors[index % colors.length];
  };

  const handleTitleNavigation = (name: string) => {
    // TODO(ganhao): 能不能让用户目录和账户目录也对齐公共目录，不要暴露具体的名称
    if (isPublicFolder(name)) {
      navigate(pathname + "/public");
    } else if (isAccountFolder(name)) {
      navigate(`${pathname}/account/${name}`);
    } else {
      navigate(`${pathname}/user/${name}`);
    }
  };

  return (
    <div>
      <PageTitle
        title="文件系统"
        description="选择一个文件夹以查看其内容，在这里对文件进行上传、下载、移动、删除等操作。"
      />
      <div
        className={cn("mt-6 grid gap-6", {
          "grid-cols-1 md:grid-cols-2": data.length === 2,
          "grid-cols-1 md:grid-cols-3": data.length === 3,
        })}
      >
        {data.map((r, index) => {
          const colors = getFolderColor(index);

          return (
            <motion.div
              key={r.name}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: index * 0.1 }}
              whileHover={{ y: -5 }}
            >
              <Card
                className={cn(
                  "group h-full transition-all duration-300",
                  "border-2 hover:shadow-lg dark:hover:shadow-primary/10",
                  colors.bg,
                  colors.border,
                  hoveredFolder === r.name && "ring-2 ring-primary/20",
                )}
                onMouseEnter={() => setHoveredFolder(r.name)}
                onMouseLeave={() => setHoveredFolder(null)}
              >
                <CardHeader className="pb-4">
                  <div className="mb-2 flex items-center gap-3">
                    {hoveredFolder === r.name ? (
                      <FolderOpen className={cn("size-6", colors.icon)} />
                    ) : (
                      <Folder className={cn("size-6", colors.icon)} />
                    )}
                    <CardTitle className="text-xl">
                      {getFolderTitle(r.name)}
                    </CardTitle>
                  </div>
                  <CardDescription className="text-balance leading-relaxed">
                    {getFolderDescription(r.name)}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div
                    className={cn(
                      "flex h-24 items-center justify-center rounded-md",
                      "bg-white/50 backdrop-blur-sm dark:bg-white/5",
                    )}
                  >
                    <div
                      className={cn(
                        "text-4xl transition-opacity",
                        "opacity-20 group-hover:opacity-30",
                        colors.icon,
                      )}
                    >
                      {index + 1}
                    </div>
                  </div>
                </CardContent>
                <CardFooter className="pt-4">
                  <Button
                    className="group w-full"
                    onClick={() => handleTitleNavigation(r.name)}
                    variant="outline"
                  >
                    <LogInIcon className="mr-2 size-4 transition-transform group-hover:translate-x-1" />
                    查看{getFolderTitle(r.name)}
                    <ChevronRight className="ml-auto size-4 transition-transform group-hover:translate-x-1" />
                  </Button>
                </CardFooter>
              </Card>
            </motion.div>
          );
        })}
      </div>

      {data.length === 0 && (
        <div className="py-12 text-center">
          <Folder className="mx-auto mb-4 size-12 text-muted-foreground/50" />
          <h3 className="mb-2 text-xl font-medium">没有找到文件夹</h3>
          <p className="text-muted-foreground">您当前没有任何文件夹。</p>
        </div>
      )}
    </div>
  );
}
