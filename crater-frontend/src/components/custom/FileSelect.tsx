import { Button } from "@/components/ui/button";
import { logger } from "@/utils/loglevel";
import { SVGProps, useState } from "react";

// 模拟API数据，根据路径获取子目录
const mockApiResponse: DirectoryItem[] = [
  {
    name: "Icon1",
  },
  {
    name: "Icon2",
  },
  {
    name: "Icon3",
  },
];
//   "src/components/pages": [
//     {
//       name: "pages",
//       children: [
//         { name: "Home.jsx" },
//         { name: "Profile.jsx" },
//         { name: "Dashboard.jsx" },
//       ],
//     },
//   ],
//   "src/components/hooks": [
//     {
//       name: "hooks",
//     },
//   ],
//   "src/components/style": [
//     {
//       name: "style",
//     },
//   ],
//   "src/components/Icons": [
//     {
//       name: "Icons",
//       children: [
//         { name: "ChevronRightIcon.jsx" },
//         { name: "FolderIcon.jsx" },
//         { name: "RefreshCwIcon.jsx" },
//       ],
//     },
//   ],
// };

// 模拟API，获取子目录
const fetchChildren = (): Promise<DirectoryItem[]> => {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve(mockApiResponse);
    }, 500); // 模拟网络延迟
  });
};

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

  const directoryPath = path ? `${path}/${name}` : name;

  const isSelected = directoryPath === selectedPath;

  const toggleOpen = () => {
    setIsOpen(!isOpen);
    if (!children && !isOpen) {
      setIsLoading(true);
      fetchChildren()
        .then((children) => {
          setChildren(children);
          setIsLoading(false);
          onPathChange(directoryPath);
        })
        .catch((e) => logger.debug(e));
    }
  };

  return (
    <div>
      <Button
        className={`w-full justify-start text-left ${isSelected ? "bg-blue-50" : ""}`}
        variant="ghost"
        onClick={toggleOpen}
      >
        {children !== undefined && (
          <ChevronRightIcon
            className={`mr-2 h-4 w-4 opacity-50 ${isOpen ? "rotate-90" : ""}`}
          />
        )}
        <FolderIcon className="mr-2 h-4 w-4 opacity-50" />
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
}

// 修改FileSelect组件初始化File数组为顶层目录
export function FileSelect({ onClose }: FileSelectProps) {
  const [selectedPath, setSelectedPath] = useState("");

  const handleRefreshClick = () => {
    setSelectedPath("");
  };

  // 顶层目录数据，仅包含第一层目录
  const topLevelDirectories = [
    { name: ".gitignore" },
    { name: ".vscode" },
    { name: "src" },
    { name: "public" },
    { name: "config" },
    { name: "package.json" },
    { name: "README.md" },
    { name: "deploy" },
    { name: ".env" },
    { name: ".env.development" },
    { name: ".env.production" },
  ];

  return (
    <div className="flex h-[400px] flex-col">
      {/* ...保留FileSelect组件的其他部分代码 */}
      <div className="flex items-center gap-2.5 border-b px-4 py-2.5">
        <ChevronRightIcon className="h-4 w-4" />
        <div className="flex-1">{selectedPath || ""}</div>
        <Button aria-label="Refresh" size="icon" onClick={handleRefreshClick}>
          <RefreshCwIcon className="h-4 w-4" />
        </Button>
      </div>
      <div className="flex-1 overflow-auto">
        <div>
          {topLevelDirectories.map((item, index) => (
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
        <Button onClick={onClose}>确认选择</Button> {/* 使用传入的 onClose */}
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
