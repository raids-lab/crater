import { ComboboxItem } from "./Combobox";
import { Dataset } from "@/services/api/dataset";

export default function DatasetItem({ item }: { item: ComboboxItem<Dataset> }) {
  return (
    <div className="text-muted-foreground flex flex-col items-start gap-0.5">
      <p className="text-foreground">{item.detail?.name}</p>
      {item.detail?.describe && (
        <p className="text-xs" data-description>
          {item.detail?.describe}
        </p>
      )}
    </div>
  );
}
