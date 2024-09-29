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
    <div className="flex items-start gap-1">
      {sortedEntries.map(([key, value]) => {
        let displayKey: string;
        if (key.includes("/")) {
          displayKey = key.split("/")[1];
        } else if (key === "memory") {
          displayKey = "mem";
        } else {
          displayKey = key;
        }
        return (
          <Badge key={key} variant="secondary" className="font-normal">
            {displayKey}: {value}
          </Badge>
        );
      })}
    </div>
  );
};

export default ResourceBadges;
