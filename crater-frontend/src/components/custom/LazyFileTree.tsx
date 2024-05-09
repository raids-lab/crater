// https://github.com/shadcn-ui/ui/issues/355
import React, { useEffect, useMemo } from "react";
import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { ChevronRight, type LucideIcon } from "lucide-react";
import useResizeObserver from "use-resize-observer";
import { useMutation, useQuery } from "@tanstack/react-query";
import { FileItem, apiGetFiles } from "@/services/api/file";

interface TreeDataItem {
  id: string;
  name: string;
  icon?: LucideIcon;
  hasChildren: boolean;
}

type TreeProps = React.HTMLAttributes<HTMLDivElement> & {
  initialSlelectedItemId?: string;
  onSelectChange?: (item: TreeDataItem | undefined) => void;
  folderIcon?: LucideIcon;
  itemIcon?: LucideIcon;
  className?: string;
};

const Tree = React.forwardRef<HTMLDivElement, TreeProps>(
  (
    {
      initialSlelectedItemId,
      onSelectChange,
      folderIcon,
      itemIcon,
      className,
      ...props
    },
    ref,
  ) => {
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
      select: (res) => res.data.data,
    });

    const { ref: refRoot, width, height } = useResizeObserver();

    return (
      <div ref={refRoot} className={cn("overflow-hidden", className)}>
        <ScrollArea style={{ width, height }}>
          <div className="relative p-2" {...props}>
            <ul>
              {data?.map((item, index) => (
                <TreeItem
                  key={index}
                  currentPath={item.name}
                  data={item}
                  ref={ref}
                  selectedItemId={selectedItemId}
                  handleSelectChange={handleSelectChange}
                  FolderIcon={folderIcon}
                  ItemIcon={itemIcon}
                />
              ))}
            </ul>
          </div>
        </ScrollArea>
      </div>
    );
  },
);
Tree.displayName = "Tree";

type TreeItemProps = TreeProps & {
  data: FileItem;
  currentPath: string;
  selectedItemId?: string;
  handleSelectChange: (item: TreeDataItem | undefined) => void;
  FolderIcon?: LucideIcon;
  ItemIcon?: LucideIcon;
};

const TreeItem = React.forwardRef<HTMLDivElement, TreeItemProps>(
  (
    {
      className,
      data,
      currentPath,
      selectedItemId,
      handleSelectChange,
      FolderIcon,
      ItemIcon,
      ...props
    },
    ref,
  ) => {
    const item: TreeDataItem = useMemo(() => {
      return {
        id: currentPath,
        name: data.name,
        icon: data.isdir ? FolderIcon : ItemIcon,
        hasChildren: data.isdir && data.size > 0,
      };
    }, [FolderIcon, ItemIcon, currentPath, data]);

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
        const children = fileList.data.data;
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
                  {item.icon && (
                    <item.icon
                      className="mr-2 h-4 w-4 shrink-0 text-accent-foreground/50"
                      aria-hidden="true"
                    />
                  )}
                  {!item.icon && FolderIcon && (
                    <FolderIcon
                      className="mr-2 h-4 w-4 shrink-0 text-accent-foreground/50"
                      aria-hidden="true"
                    />
                  )}
                  <span className="truncate text-sm">{item.name}</span>
                </AccordionTrigger>
                <AccordionContent className="pl-6">
                  <ul>
                    {children.map((data, index) => (
                      <TreeItem
                        key={index}
                        data={data}
                        currentPath={`${currentPath}/${data.name}`}
                        selectedItemId={selectedItemId}
                        handleSelectChange={handleSelectChange}
                        FolderIcon={FolderIcon}
                        ItemIcon={ItemIcon}
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
              Icon={ItemIcon}
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
    Icon?: LucideIcon;
    className?: string;
  }
>(({ className, item, isSelected, Icon, ...props }, ref) => {
  return (
    <div
      ref={ref}
      className={cn(
        "flex cursor-pointer items-center px-2 py-2         before:absolute before:left-0 before:right-1 before:-z-10 before:h-[1.75rem] before:w-full before:bg-muted/80 before:opacity-0 hover:before:opacity-100",
        className,
        isSelected &&
          "text-accent-foreground before:border-l-2 before:border-l-accent-foreground/50 before:bg-accent before:opacity-100 dark:before:border-0",
      )}
      {...props}
    >
      {item.icon && (
        <item.icon
          className="mr-2 h-4 w-4 shrink-0 text-accent-foreground/50"
          aria-hidden="true"
        />
      )}
      {!item.icon && Icon && (
        <Icon
          className="mr-2 h-4 w-4 shrink-0 text-accent-foreground/50"
          aria-hidden="true"
        />
      )}
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
