import { cn } from "@/lib/utils";
import { Badge } from "../ui/badge";
import { NodeType } from "@/services/api/node";

export const nodeTypes = [
  {
    value: "hygon",
    label: "Hygon",
  },
  {
    value: "shenwei",
    label: "Shenwei",
  },
  {
    value: "yitian",
    label: "Yitian",
  },
];

const getNodeTypeLabel = (
  phase: NodeType,
): {
  label: string;
  color: string;
} => {
  switch (phase) {
    case NodeType.Hygon:
      return {
        label: "海光",
        color: "text-purple-600 bg-purple-500/10",
      };
    case NodeType.Shenwei:
      return {
        label: "申威",
        color: "text-amber-600 bg-amber-500/10",
      };
    case NodeType.Yitian:
      return {
        label: "倚天",
        color: "text-cyan-600 bg-cyan-500/10",
      };
    default:
      return {
        label: "Intel",
        color: "text-slate-500 bg-slate-500/10",
      };
  }
};

const NodeTypeLabel = ({ nodeType }: { nodeType: NodeType }) => {
  const data = getNodeTypeLabel(nodeType);

  return (
    <Badge className={cn("border-none", data.color)} variant="outline">
      <div className="">{data.label}</div>
    </Badge>
  );
};

export default NodeTypeLabel;
