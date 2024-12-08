// https://github.com/shadcn-ui/ui/issues/355
import React, { useEffect, useMemo } from "react";
import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import {
  ChevronRight,
  FileDigitIcon,
  FolderIcon,
  type LucideIcon,
} from "lucide-react";
import useResizeObserver from "use-resize-observer";
import { useMutation, useQuery } from "@tanstack/react-query";
import { FileItem, apiGetFiles } from "@/services/api/file";

interface TreeDataItem {
  id: string;
  name: string;
  icon: LucideIcon;
  hasChildren: boolean;
}

export const getFolderTitle = (folder: string) => {
  // public: 公共空间
  // q-*: 账户共享空间
  // u-*: 用户私有空间
  if (folder === "public") {
    return "公共空间";
  } else if (folder.startsWith("q-")) {
    return "账户空间";
  }
  return "用户空间";
};

type TreeProps = React.HTMLAttributes<HTMLDivElement> & {
  initialSlelectedItemId?: string;
  onSelectChange?: (item: TreeDataItem | undefined) => void;
  className?: string;
};

const Tree = React.forwardRef<HTMLDivElement, TreeProps>(
  ({ initialSlelectedItemId, onSelectChange, className, ...props }, ref) => {
    const [selectedItemId, setSelectedItemId] = React.useState<
      string | undefined
    >(initialSlelectedItemId);

    const handleSelectChange = React.useCallback(
      (item: TreeDataItem | undefined) => {
        setSelectedItemId(item?.id);
        if (onSelectChange) {
          onSelectChange(item);
        }
      },
      [onSelectChange],
    );

    const { data } = useQuery({
      queryKey: ["directory", "list"],
      queryFn: () => apiGetFiles(""),
      select: (res) =>
        res.data.data?.sort((a, b) => {
          return a.name.localeCompare(b.name);
        }) ?? [],
    });

    const { ref: refRoot, width, height } = useResizeObserver();

    return (
      <div ref={refRoot} className={cn("overflow-hidden", className)}>
        <ScrollArea style={{ width, height }}>
          <div className="relative p-2" {...props}>
            <ul>
              {data?.map((item, index) => (
                <TreeItem
                  level={0}
                  key={index}
                  currentPath={item.name}
                  data={item}
                  ref={ref}
                  selectedItemId={selectedItemId}
                  handleSelectChange={handleSelectChange}
                />
              ))}
            </ul>
          </div>
          <ScrollBar orientation="horizontal" />
        </ScrollArea>
      </div>
    );
  },
);
Tree.displayName = "Tree";

type TreeItemProps = TreeProps & {
  data: FileItem;
  level: number;
  currentPath: string;
  selectedItemId?: string;
  handleSelectChange: (item: TreeDataItem | undefined) => void;
};

const TreeItem = React.forwardRef<HTMLDivElement, TreeItemProps>(
  (
    {
      level,
      data,
      currentPath,
      selectedItemId,
      handleSelectChange,
      className,
      ...props
    },
    ref,
  ) => {
    const item: TreeDataItem = useMemo(() => {
      return {
        id: currentPath,
        name: level === 0 ? getFolderTitle(data.name) : data.name,
        icon: data.isdir ? FolderIcon : FileDigitIcon,
        hasChildren: data.isdir && data.size > 0,
      };
    }, [currentPath, data, level]);

    useEffect(() => {
      if (data.isdir && data.size > 0) {
        // create default childrens with length of data.size
        const newChildren: FileItem[] = Array.from(
          { length: data.size },
          () => ({
            isdir: false,
            name: "",
            size: 0,
            modifytime: "",
          }),
        );
        setChildren(newChildren);
      }
    }, [data]);

    const [children, setChildren] = React.useState<FileItem[]>([]);
    const [childrenInitialized, setChildrenInitialized] = React.useState(false);

    const { mutate: getChildren } = useMutation({
      mutationFn: () => apiGetFiles(currentPath),
      onSuccess: (fileList) => {
        const children =
          fileList.data.data?.sort((a, b) => {
            if (a.isdir && !b.isdir) {
              return -1; // a在b之前
            } else if (!a.isdir && b.isdir) {
              return 1; // a在b之后
            } else {
              return a.name.localeCompare(b.name);
            }
          }) ?? [];
        if (children) {
          setChildren(children);
          setChildrenInitialized(true);
        }
      },
    });

    return (
      <li>
        <div ref={ref} role="tree" className={className} {...props}>
          {item.hasChildren ? (
            <AccordionPrimitive.Root type="multiple">
              <AccordionPrimitive.Item value={item.id}>
                <AccordionTrigger
                  className={cn(
                    "px-2 before:absolute before:left-0 before:-z-10 before:h-[1.75rem] before:w-full before:bg-muted/80 before:opacity-0 hover:before:opacity-100",
                    selectedItemId === item.id &&
                      "text-accent-foreground before:border-l-2 before:border-l-accent-foreground/50 before:bg-accent before:opacity-100 dark:before:border-0",
                  )}
                  onClick={() => {
                    if (!childrenInitialized) {
                      getChildren();
                    }
                    handleSelectChange(item);
                  }}
                >
                  <item.icon
                    className="mr-2 h-4 w-4 shrink-0 text-accent-foreground/50"
                    aria-hidden="true"
                  />
                  <span className="truncate text-sm">{item.name}</span>
                </AccordionTrigger>
                <AccordionContent className="pl-6">
                  <ul>
                    {children.map((data, index) => (
                      <TreeItem
                        level={level + 1}
                        key={index}
                        data={data}
                        currentPath={`${currentPath}/${data.name}`}
                        selectedItemId={selectedItemId}
                        handleSelectChange={handleSelectChange}
                      />
                    ))}
                  </ul>
                </AccordionContent>
              </AccordionPrimitive.Item>
            </AccordionPrimitive.Root>
          ) : (
            <Leaf
              item={item}
              isSelected={selectedItemId === item.id}
              onClick={() => handleSelectChange(item)}
            />
          )}
        </div>
      </li>
    );
  },
);
TreeItem.displayName = "TreeItem";

const Leaf = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & {
    item: TreeDataItem;
    isSelected?: boolean;
    className?: string;
  }
>(({ className, item, isSelected, ...props }, ref) => {
  return (
    <div
      ref={ref}
      className={cn(
        "flex cursor-pointer items-center px-2 py-2 before:absolute before:left-0 before:right-1 before:-z-10 before:h-[1.75rem] before:w-full before:bg-muted/80 before:opacity-0 hover:before:opacity-100",
        className,
        isSelected &&
          "text-accent-foreground before:border-l-2 before:border-l-accent-foreground/50 before:bg-accent before:opacity-100 dark:before:border-0",
      )}
      {...props}
    >
      <item.icon
        className="mr-2 h-4 w-4 shrink-0 text-accent-foreground/50"
        aria-hidden="true"
      />
      <span className="flex-grow truncate text-sm">{item.name}</span>
    </div>
  );
});
Leaf.displayName = "Leaf";

const AccordionTrigger = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Trigger> & {
    className?: string;
  }
>(({ className, children, ...props }, ref) => (
  <AccordionPrimitive.Header>
    <AccordionPrimitive.Trigger
      ref={ref}
      className={cn(
        "flex w-full flex-1 items-center py-2 transition-all last:[&[data-state=open]>svg]:rotate-90",
        className,
      )}
      {...props}
    >
      {children}
      <ChevronRight className="ml-auto h-4 w-4 shrink-0 text-accent-foreground/50 transition-transform duration-200" />
    </AccordionPrimitive.Trigger>
  </AccordionPrimitive.Header>
));
AccordionTrigger.displayName = AccordionPrimitive.Trigger.displayName;

const AccordionContent = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Content> & {
    className?: string;
  }
>(({ className, children, ...props }, ref) => (
  <AccordionPrimitive.Content
    ref={ref}
    className={cn(
      "overflow-hidden text-sm transition-all data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down",
      className,
    )}
    {...props}
  >
    <div className="pb-0 pt-0">{children}</div>
  </AccordionPrimitive.Content>
));
AccordionContent.displayName = AccordionPrimitive.Content.displayName;

export { Tree, type TreeDataItem };
