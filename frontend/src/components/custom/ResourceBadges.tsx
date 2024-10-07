import { Badge } from "@/components/ui/badge";

const ResourceBadges = ({
  resources,
}: {
  resources: Record<string, string>;
}) => {
  const sortedEntries = Object.entries(resources).sort(([keyA], [keyB]) => {
    if (keyA === "cpu") return -1;
    if (keyB === "cpu") return 1;
    if (keyA === "memory") return keyB === "cpu" ? 1 : -1;
    if (keyB === "memory") return keyA === "cpu" ? -1 : 1;
    return keyA.localeCompare(keyB);
  });

  return (
    <div className="flex flex-col items-start gap-1 lg:flex-row lg:flex-wrap">
      {sortedEntries.map(([key, value]) => {
        let display: string;
        if (key.includes("/")) {
          display = `${key.split("/")[1]}: ${value}`;
        } else if (key === "memory") {
          display = `mem: ${value}`;
        } else if (key === "cpu") {
          display = `cpu: ${value}`;
        } else {
          display = `${key}: ${value}`;
        }
        return (
          <Badge key={key} variant="secondary" className="font-normal">
            {display}
          </Badge>
        );
      })}
    </div>
  );
};

export default ResourceBadges;
