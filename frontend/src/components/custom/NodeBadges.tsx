import { Badge } from "@/components/ui/badge";

const NodeBadges = ({ nodes }: { nodes: string[] }) => {
  if (nodes?.length > 1) {
    return (
      <Badge variant="secondary" className="font-normal">
        {nodes.length} 节点
      </Badge>
    );
  }

  return (
    <div className="flex items-start gap-1">
      {nodes?.map((node) => (
        <Badge key={node} variant="secondary" className="font-normal">
          {node}
        </Badge>
      ))}
    </div>
  );
};

export default NodeBadges;
