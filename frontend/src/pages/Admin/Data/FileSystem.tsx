import { FileSystemTable } from "@/components/file/FileSystemTable";
import { apiGetAdminFile } from "@/services/api/file";

export const Component = () => {
  return <FileSystemTable apiGetFiles={apiGetAdminFile} isadmin={true} />;
};
