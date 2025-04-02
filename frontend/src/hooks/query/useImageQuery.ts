import { ComboboxItem } from "@/components/form/Combobox";
import { ImageInfoResponse } from "@/services/api/imagepack";
import { apiJTaskImageList, JobType } from "@/services/api/vcjob";
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
      )
        // Sort by creation time, newest first
        .sort((a, b) => {
          // Adjust the field name if needed based on your data structure
          return (
            new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
          );
        })
        .map(
          (item) =>
            ({
              value: item.imageLink,
              label: item.description,
              detail: item,
            }) as ComboboxItem<ImageInfoResponse>,
        );
      return items;
    },
  });
};

export default useImageQuery;
