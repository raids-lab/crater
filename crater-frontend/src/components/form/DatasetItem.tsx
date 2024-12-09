import { ComboboxItem } from "./Combobox";
import { Dataset } from "@/services/api/dataset";

export default function DatasetItem({ item }: { item: ComboboxItem<Dataset> }) {
  return (
    <div className="flex flex-col items-start gap-0.5 text-muted-foreground">
      <p className="text-foreground">{item.detail?.name}</p>
      {item.detail?.describe && (
        <p className="text-xs" data-description>
          {item.detail?.describe}
        </p>
      )}
    </div>
  );
}
