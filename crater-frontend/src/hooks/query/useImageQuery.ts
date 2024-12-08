import { ComboboxItem } from "@/components/form/Combobox";
import { ImageInfoResponse } from "@/services/api/imagepack";
import { apiJTaskImageList, JobType } from "@/services/api/vcjob";
import { shortenImageName } from "@/utils/formatter";
import { useQuery } from "@tanstack/react-query";

const useImageQuery = (type?: JobType) => {
  return useQuery({
    queryKey: ["images", type],
    queryFn: () => apiJTaskImageList(type || JobType.Jupyter),
    select: (res) => {
      const items = Array.from(
        new Map(
          res.data.data.images.map((item) => [item.imageLink, item]),
        ).values(),
      ).map(
        (item) =>
          ({
            value: item.imageLink,
            label: shortenImageName(item.imageLink),
            detail: item,
          }) as ComboboxItem<ImageInfoResponse>,
      );
      return items;
    },
  });
};

export default useImageQuery;
