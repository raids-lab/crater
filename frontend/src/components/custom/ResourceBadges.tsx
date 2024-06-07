import { Badge } from "@/components/ui/badge";

const ResourceBadges = ({
  resources,
}: {
  resources: Record<string, string>;
}) => {
  return (
    <div className="flex items-start gap-1">
      {Object.entries(resources).map(([key, value]) => (
        <Badge key={key} variant="secondary" className="font-normal">
          {key.replace("nvidia.com/", "").replace("memory", "mem")}: {value}
        </Badge>
      ))}
    </div>
  );
};

export default ResourceBadges;
