import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { ChevronDownIcon, PlusCircleIcon } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { useNavigate } from "react-router-dom";
import { useLocalStorage } from "usehooks-ts";

interface URL {
  url: string;
  name: string;
  disabled?: boolean;
}

const SplitButton = ({ urls, title }: { urls: URL[]; title: string }) => {
  const [position, setPosition] = useLocalStorage(
    `split-button-${title}`,
    urls[0].url,
  );
  const navigate = useNavigate();
  return (
    <div className="flex w-fit items-center space-x-1 rounded-md bg-primary text-primary-foreground">
      <Button
        className="pr-3 shadow-none"
        onClick={() => navigate(`/${position}`)}
      >
        <PlusCircleIcon className="-ml-0.5 mr-1.5 h-4 w-4" />
        新建{urls.find((url) => url.url === position)?.name}
      </Button>
      <Separator orientation="vertical" className="h-[20px]" />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button className="pl-2 pr-3 shadow-none focus:ring-0">
            <ChevronDownIcon className="h-4 w-4 text-primary-foreground" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          alignOffset={-5}
          className="w-[200px]"
          forceMount
        >
          <DropdownMenuLabel>作业类型</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuRadioGroup value={position} onValueChange={setPosition}>
            {urls.map((url) => (
              <DropdownMenuRadioItem
                key={url.url}
                value={url.url}
                disabled={url.disabled}
              >
                {url.name}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
};

export default SplitButton;
