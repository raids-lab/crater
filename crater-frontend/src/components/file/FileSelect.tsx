import { Button } from "@/components/ui/button";
import { SVGProps, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { apiGetFiles } from "@/services/api/file";
import { useEffect, useMemo } from "react";

interface DirectoryItem {
  name: string;
  children?: DirectoryItem[]; // 可选属性，存在于文件夹中
}

type PathChangeHandler = (path: string) => void;

interface DirectoryProps {
  name: string;
  path: string;
  onPathChange: PathChangeHandler;
  selectedPath: string;
  //top: { [key: string]: string }[];
}

function LoadingIndicator() {
  return (
    <div className="bg-whit absolute inset-0 flex items-center justify-center bg-opacity-50">
      <div className="flex items-center space-x-3 rounded-md p-3 text-blue-300">
        <svg
          className="h-8 w-8 animate-spin text-blue-300"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          ></circle>
          <path
            className="opacity-75"
            fill="currentColor"
            d="M22 12A10 10 0 0 1 12 22V12z"
          ></path>
        </svg>
        <span className="text-lg">Loading...</span>
      </div>
    </div>
  );
}

function Directory({
  name,
  path = "",
  onPathChange,
  selectedPath,
}: DirectoryProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [children, setChildren] = useState<DirectoryItem[] | undefined>(
    undefined,
  );
  const [isLoading, setIsLoading] = useState(false);

  const [directoryPath, truePath, isSelected] = useMemo(() => {
    const directoryPath = path ? `${path}/${name}` : `/${name}`;
    // const parts = directoryPath.split("/");
    // const leavePath = directoryPath.replace(`/${parts[1]}`, ""); //只会替换第一次出现
    // const truePath = "/" + name + leavePath;
    const truePath = directoryPath;
    const isSelected = directoryPath === selectedPath;
    return [directoryPath, truePath, isSelected];
  }, [path, name, selectedPath]);

  const { mutate: getDirectoryList } = useMutation({
    mutationFn: (truePath: string) => apiGetFiles(truePath),
    onSuccess: (fileList) => {
      if (fileList.data.data !== null) {
        const childDirectories: DirectoryItem[] =
          fileList.data.data
            ?.filter((file) => file.isdir)
            .map((file) => {
              return {
                name: file.name,
              };
            }) ?? [];
        setIsLoading(false);
        onPathChange(directoryPath);
        setChildren(childDirectories);
      } else {
        setIsLoading(false);
        onPathChange(directoryPath);
        setChildren(undefined);
      }
    },
  });

  const toggleOpen = () => {
    setIsOpen(!isOpen);
    if (!children && !isOpen) {
      setIsLoading(true);
      getDirectoryList(truePath);
    }
  };

  return (
    <div>
      <Button
        className={`left-0 w-[500px] justify-start px-1 py-2.5 text-left ${isSelected ? "bg-blue-50" : ""}`}
        variant="ghost"
        onClick={toggleOpen}
      >
        <ChevronRightIcon
          className={`mr-2 size-4 opacity-50 ${isOpen ? "rotate-90" : ""}`}
        />
        <FolderIcon className="mr-2 size-4 text-sky-600 opacity-50" />
        {name}
      </Button>
      {isLoading && <LoadingIndicator />} {/* 使用自定义LoadingIndicator组件 */}
      {isOpen && children && (
        <div className="pl-4">
          {children.map((child, index) => (
            <Directory
              key={index}
              name={child.name}
              path={directoryPath}
              onPathChange={onPathChange}
              selectedPath={selectedPath}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// TaskFormProps扩充HTMLDivElement属性，增加onClose函数
interface FileSelectProps extends React.HTMLAttributes<HTMLDivElement> {
  onClose: () => void;
  handleSubpathInfo: (info: string) => void;
  handleSubpath: (info: string) => void;
}

// 修改FileSelect组件初始化File数组为顶层目录
export function FileSelect({
  onClose,
  handleSubpathInfo,
  handleSubpath,
}: FileSelectProps) {
  const [selectedPath, setSelectedPath] = useState("");

  const [topLevelDirectorieList, setTopLevelDirectorieList] = useState<
    DirectoryItem[] | undefined
  >(undefined);

  // const [top, setTop] = useState<{ [key: string]: string }[]>([]);

  const handleRefreshClick = () => {
    setSelectedPath("");
  };

  const { data: FileList, isLoading } = useQuery({
    queryKey: ["directory", "list"],
    queryFn: () => apiGetFiles("/"),
    select: (res) => res.data.data,
  });

  useEffect(() => {
    if (isLoading) return;
    if (!FileList) return;
    const topLevelDirectories: DirectoryItem[] = FileList.map((file) => {
      return {
        name: file.name,
      };
    });

    setTopLevelDirectorieList(topLevelDirectories);
  }, [FileList, isLoading]);

  return (
    <div className="left-0 flex h-[410px] w-[590px] max-w-screen-sm flex-col">
      {/* ...保留FileSelect组件的其他部分代码 */}
      <div className="bg-slate-150 left-0 flex h-[50px] w-full items-center gap-2.5 rounded-lg border-2 px-1 py-2.5">
        <ChevronRightIcon className="size-4" />
        <div className="flex-1">{selectedPath || ""}</div>
        <Button aria-label="Refresh" size="icon" onClick={handleRefreshClick}>
          <RefreshCwIcon className="size-4" />
        </Button>
      </div>
      <div className="w-[590px] flex-1 overflow-auto">
        <div>
          {topLevelDirectorieList &&
            topLevelDirectorieList.map((item, index) => (
              <Directory
                key={index}
                name={item.name}
                path=""
                onPathChange={setSelectedPath}
                selectedPath={selectedPath} // 将 selectedPath 传递给 Directory 组件
              />
            ))}
        </div>
      </div>
      {/* ...其余代码 */}
      <div className="flex justify-end p-4">
        <Button
          onClick={() => {
            onClose();
            handleSubpathInfo(selectedPath);
            handleSubpath(selectedPath);
          }}
        >
          确认选择
        </Button>{" "}
        {/* 使用传入的 onClose */}
      </div>
    </div>
  );
}

function ChevronRightIcon(
  props: JSX.IntrinsicAttributes & SVGProps<SVGSVGElement>,
) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="m9 18 6-6-6-6" />
    </svg>
  );
}

function FolderIcon(props: JSX.IntrinsicAttributes & SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z" />
    </svg>
  );
}

function RefreshCwIcon(
  props: JSX.IntrinsicAttributes & SVGProps<SVGSVGElement>,
) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
      <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
      <path d="M8 16H3v5" />
    </svg>
  );
}
