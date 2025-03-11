import { shortenImageName } from "@/utils/formatter";

const ImageLabel = ({
  description,
  url,
}: {
  description: string;
  url: string;
}) => {
  return (
    <div className="flex flex-col gap-0.5 text-left">
      <span className="truncate text-sm font-normal">{description}</span>
      <span className="text-muted-foreground truncate font-mono text-xs">
        {shortenImageName(url)}
      </span>
    </div>
  );
};

export default ImageLabel;
