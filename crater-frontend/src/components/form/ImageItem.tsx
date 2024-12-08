import { cn } from "@/lib/utils";
import { ImageInfoResponse } from "@/services/api/imagepack";
import { ComboboxItem } from "./Combobox";
import { BoxIcon } from "lucide-react";
import { shortenImageName } from "@/utils/formatter";

export default function ImageItem({
  item,
}: {
  item: ComboboxItem<ImageInfoResponse>;
}) {
  return (
    <div className="flex items-center gap-3 text-muted-foreground">
      <div
        className={cn(
          "flex size-8 items-center justify-center rounded-full font-normal",
          {
            "bg-primary/15 text-primary": item.detail?.isPublic,
            "bg-purple-500/15 text-purple-500": !item.detail?.isPublic,
          },
        )}
      >
        <BoxIcon className="size-5" />
      </div>
      <div className="flex flex-col items-start gap-0.5">
        {item.detail?.description && (
          <p className="text-foreground">{item.detail?.description}</p>
        )}
        <p className="truncate font-mono text-xs" data-description>
          {shortenImageName(item.detail?.imageLink ?? "")}
        </p>
      </div>
    </div>
  );
}
