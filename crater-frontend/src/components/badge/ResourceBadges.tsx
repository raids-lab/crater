import { useState, useEffect } from "react";
import { Badge } from "@/components/ui/badge";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverClose,
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

export default function ResourceBadges({
  namespace,
  podName,
  resources = {},
  showEdit,
}: ResourceBadgesProps) {
  const isAdmin = useIsAdmin();
  const queryClient = useQueryClient();
  const [localResources, setLocalResources] =
    useState<Record<string, string>>(resources);
  const [editingValues, setEditingValues] = useState<Record<string, string>>(
    {},
  );

  useEffect(() => {
    setLocalResources(resources);
  }, [resources]);

  type UpdateResourceVars = {
    namespace: string;
    podName: string;
    key: "cpu" | "memory";
    numericValue: string;
  };

  const { mutate: updateResource } = useMutation<
    AxiosResponse<IResponse<string>>,
    Error,
    UpdateResourceVars
  >({
    mutationFn: ({ podName, key, numericValue }) =>
      apiAdminResourceReset(namespace, podName, key, numericValue),
    onSuccess: (_res, { podName, key, numericValue }) => {
      queryClient.invalidateQueries({ queryKey: ["podResources", podName] });
      setLocalResources((prev) => ({
        ...prev,
        [key]: numericValue,
      }));
      toast.success(
        `${namespace} ${podName} ${key} updated to ${numericValue}`,
      );
    },
    onError: (_err, { podName, key }) => {
      toast.error(`Failed to update ${key} for ${podName}`);
    },
  });

  const updateResourceHandler = (
    namespace: string | undefined,
    podName: string | undefined,
    key: "cpu" | "memory",
    numericValue: string,
  ) => {
    if (podName && namespace) {
      updateResource({ namespace, podName, key, numericValue });
    }
  };

  // Sort cpu first, then memory, then alphabet
  const sortedEntries = Object.entries(localResources).sort(([a], [b]) => {
    if (a === "cpu") return -1;
    if (b === "cpu") return 1;
    if (a === "memory") return b === "cpu" ? 1 : -1;
    if (b === "memory") return a === "cpu" ? -1 : 1;
    return a.localeCompare(b);
  });

  return (
    <div className="flex flex-col flex-wrap gap-1 lg:flex-row">
      {sortedEntries.map(([rawKey, rawValue]) => {
        const key = rawKey.includes("/")
          ? rawKey.split("/").slice(1).join("")
          : rawKey;
        const editable =
          showEdit && isAdmin && (key === "cpu" || key === "memory");
        const display =
          key === "cpu"
            ? `${rawValue}c`
            : key === "memory"
              ? `${rawValue}`
              : `${key}: ${rawValue}`;

        if (!editable) {
          return (
            <Badge key={key} variant="secondary" className="font-mono">
              {display}
            </Badge>
          );
        }

        return (
          <Popover key={key}>
            <PopoverTrigger asChild>
              <Badge
                className="cursor-pointer font-mono select-none hover:bg-gray-200"
                variant="secondary"
              >
                {display}
              </Badge>
            </PopoverTrigger>
            <PopoverContent className="w-64 space-y-3 p-4">
              <h4 className="font-medium">Configure {key.toUpperCase()}</h4>
              <div className="flex items-center gap-2">
                <Input
                  type="text"
                  defaultValue={rawValue}
                  onChange={(e) =>
                    setEditingValues((prev) => ({
                      ...prev,
                      [key]: e.target.value,
                    }))
                  }
                  className="flex-1"
                />
              </div>
              <PopoverClose asChild>
                <Button
                  size="sm"
                  className="w-full"
                  disabled={!(editingValues[key] ?? rawValue)}
                  onClick={() => {
                    const newVal = editingValues[key] ?? rawValue;
                    updateResourceHandler(
                      namespace,
                      podName,
                      key as "cpu" | "memory",
                      newVal,
                    );
                  }}
                >
                  Save
                </Button>
              </PopoverClose>
            </PopoverContent>
          </Popover>
        );
      })}
    </div>
  );
}
