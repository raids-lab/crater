import { useState, useEffect, useMemo, useCallback } from "react";
import { Badge } from "@/components/ui/badge";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import useIsAdmin from "@/hooks/useAdmin";
import { apiAdminResourceReset } from "@/services/api/resource";
import { toast } from "sonner";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";

interface ResourceBadgesProps {
  namespace?: string;
  podName?: string;
  resources?: Record<string, string>;
  showEdit?: boolean;
}

type UpdateResourceVars = {
  namespace: string;
  podName: string;
  key: "cpu" | "memory";
  numericValue: string;
};

// 提取单独的 ResourceBadge 子组件，提高渲染效率
const ResourceBadge = ({
  keyName,
  value,
  editable,
  onUpdate,
}: {
  keyName: string;
  value: string;
  editable: boolean;
  namespace?: string;
  podName?: string;
  onUpdate: (key: "cpu" | "memory", value: string) => void;
}) => {
  const [editValue, setEditValue] = useState(value);

  useEffect(() => {
    setEditValue(value);
  }, [value]);

  const display =
    keyName === "cpu"
      ? `${value}c`
      : keyName === "memory"
        ? `${value}`
        : `${keyName}: ${value}`;

  if (!editable) {
    return (
      <Badge variant="secondary" className="font-mono">
        {display}
      </Badge>
    );
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Badge
          className="hover:bg-primary hover:text-primary-foreground cursor-pointer font-mono select-none"
          variant="secondary"
          title="Click to edit resource"
        >
          {display}
        </Badge>
      </PopoverTrigger>
      <PopoverContent className="w-64 space-y-3 p-4">
        <h4 className="font-medium">Configure {keyName.toUpperCase()}</h4>
        <div className="flex items-center gap-2">
          <Input
            type="text"
            value={editValue}
            onChange={(e) => setEditValue(e.target.value)}
            className="flex-1"
          />
        </div>
        <Button
          size="sm"
          className="w-full"
          disabled={!editValue}
          onClick={() => onUpdate(keyName as "cpu" | "memory", editValue)}
        >
          Save
        </Button>
      </PopoverContent>
    </Popover>
  );
};

export default function ResourceBadges({
  namespace,
  podName,
  resources = {},
  showEdit = false,
}: ResourceBadgesProps) {
  const isAdmin = useIsAdmin();
  const queryClient = useQueryClient();

  const { mutate: updateResource } = useMutation<
    AxiosResponse<IResponse<string>>,
    Error,
    UpdateResourceVars
  >({
    mutationFn: ({ namespace, podName, key, numericValue }) =>
      apiAdminResourceReset(namespace, podName, key, numericValue),
    onSuccess: (_res, { podName, key, numericValue }) => {
      queryClient.invalidateQueries({ queryKey: ["podResources", podName] });
      toast.success(
        `${namespace} ${podName} ${key} updated to ${numericValue}`,
      );
    },
    onError: (_err, { podName, key }) => {
      toast.error(`Failed to update ${key} for ${podName}`);
    },
  });

  // 使用 useCallback 缓存更新函数
  const handleUpdateResource = useCallback(
    (key: "cpu" | "memory", value: string) => {
      if (podName && namespace) {
        updateResource({ namespace, podName, key, numericValue: value });
      }
    },
    [namespace, podName, updateResource],
  );

  // 使用 useMemo 缓存排序结果
  const sortedEntries = useMemo(
    () =>
      Object.entries(resources).sort(([a], [b]) => {
        if (a === "cpu") return -1;
        if (b === "cpu") return 1;
        if (a === "memory") return b === "cpu" ? 1 : -1;
        if (b === "memory") return a === "cpu" ? -1 : 1;
        return a.localeCompare(b);
      }),
    [resources],
  );

  return (
    <div className="flex flex-col flex-wrap gap-1 lg:flex-row">
      {sortedEntries.map(([rawKey, rawValue]) => {
        const key = rawKey.includes("/")
          ? rawKey.split("/").slice(1).join("")
          : rawKey;
        const editable =
          showEdit && isAdmin && (key === "cpu" || key === "memory");

        return (
          <ResourceBadge
            key={key}
            keyName={key}
            value={rawValue}
            editable={editable}
            namespace={namespace}
            podName={podName}
            onUpdate={handleUpdateResource}
          />
        );
      })}
    </div>
  );
}
