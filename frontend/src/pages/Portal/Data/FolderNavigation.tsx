import { useState } from "react";
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

const getFolderDescription = (folder: string) => {
  // public: 公共空间
  // q-*: 账户共享空间
  // u-*: 用户私有空间
  if (folder === "sugon-gpu-incoming") {
    return "平台上所有用户都可以访问，如需获取上传权限，请联系管理员。";
  } else if (folder.startsWith("q-")) {
    return "账户共享空间，仅限账户内用户访问，如需获取上传权限，请联系账户管理员。";
  }
  return "个人空间，仅限当前用户访问，拥有完全的读写权限。";
};

export default function FolderNavigation({ data }: { data?: FileItem[] }) {
  const [hoveredFolder, setHoveredFolder] = useState<string | null>(null);
  const { pathname } = useLocation();
  const navigate = useNavigate();

  // 为每个文件夹分配一个固定的颜色
  const getFolderColor = (index: number) => {
    const colors = [
      "bg-blue-50",
      "bg-green-50",
      "bg-purple-50",
      "bg-amber-50",
      "bg-rose-50",
    ];
    const borderColors = [
      "border-blue-200",
      "border-green-200",
      "border-purple-200",
      "border-amber-200",
      "border-rose-200",
    ];
    const iconColors = [
      "text-blue-500",
      "text-green-500",
      "text-purple-500",
      "text-amber-500",
      "text-rose-500",
    ];

    return {
      bg: colors[index % colors.length],
      border: borderColors[index % borderColors.length],
      icon: iconColors[index % iconColors.length],
    };
  };

  const handleTitleNavigation = (name: string) => {
    if (name == "sugon-gpu-incoming") {
      navigate(pathname + "/public");
    } else if (name.startsWith("q-")) {
      navigate(`${pathname}/account/${name}`);
    } else {
      navigate(`${pathname}/user/${name}`);
    }
  };

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8 text-center">
        <h1 className="mb-3 text-3xl font-bold tracking-tight">我的文件夹</h1>
        <p className="mx-auto max-w-2xl text-muted-foreground">
          选择一个文件夹以查看其内容。这些文件夹包含了您所有的重要资料。
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-2 md:gap-8 lg:grid-cols-3">
        {data?.map((r, index) => {
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
                className={`h-full transition-all duration-300 hover:shadow-lg ${
                  hoveredFolder === r.name ? "ring-2 ring-primary/20" : ""
                } ${colors.bg} border-2 ${colors.border}`}
                onMouseEnter={() => setHoveredFolder(r.name)}
                onMouseLeave={() => setHoveredFolder(null)}
              >
                <CardHeader className="pb-4">
                  <div className="mb-2 flex items-center gap-3">
                    {hoveredFolder === r.name ? (
                      <FolderOpen className={`size-6 ${colors.icon}`} />
                    ) : (
                      <Folder className={`size-6 ${colors.icon}`} />
                    )}
                    <CardTitle className="text-xl">
                      {getFolderTitle(r.name)}
                    </CardTitle>
                  </div>
                  <CardDescription className="text-balance leading-relaxed">
                    {getFolderDescription(r.name)}
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex-grow">
                  <div className="flex h-24 items-center justify-center rounded-md bg-white/50">
                    <div className={`text-4xl opacity-20 ${colors.icon}`}>
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

      {(!data || data.length === 0) && (
        <div className="py-12 text-center">
          <Folder className="mx-auto mb-4 size-12 text-muted-foreground/50" />
          <h3 className="mb-2 text-xl font-medium">没有找到文件夹</h3>
          <p className="text-muted-foreground">
            您当前没有任何文件夹。请创建一个新文件夹开始使用。
          </p>
        </div>
      )}
    </div>
  );
}
