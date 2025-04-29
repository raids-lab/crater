import { shortenImageName } from "@/utils/formatter";
import { Badge } from "@/components/ui/badge";
const ImageLabel = ({
  description,
  url,
  tags = [],
}: {
  description: string;
  url: string;
  tags: string[];
}) => {
  return (
    <div className="flex flex-col gap-0.5 text-left">
      <span className="truncate text-sm font-normal">{description}</span>
      <span className="text-muted-foreground truncate font-mono text-xs">
        {shortenImageName(url)}
      </span>
      <span className="text-muted-foreground truncate font-mono text-xs">
        <div className="flex max-w-[500px] flex-wrap gap-1">
          {tags != null &&
            tags.length !== 0 &&
            tags.map((tag) => (
              <Badge variant="secondary" key={tag} className="rounded-full">
                {tag}
              </Badge>
            ))}
        </div>
      </span>
    </div>
  );
};

export default ImageLabel;
