import { FileSystemTable } from "@/components/file/FileSystemTable";
import { apiGetFiles } from "@/services/api/file";

export const Component = () => {
  return <FileSystemTable apiGetFiles={apiGetFiles} isadmin={false} />;
};
