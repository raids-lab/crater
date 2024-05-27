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
import { useState } from "react";
import { useNavigate } from "react-router-dom";

interface URL {
  url: string;
  name: string;
}

const SplitButton = ({
  urls,
  defaultValue,
}: {
  urls: URL[];
  defaultValue: string;
}) => {
  const [position, setPosition] = useState(defaultValue);
  const navigate = useNavigate();
  return (
    <div className="flex w-fit items-center space-x-1 rounded-md bg-primary text-primary-foreground">
      <Button
        className="pr-3 shadow-none"
        onClick={() => navigate(`new-${position}`)}
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
              <DropdownMenuRadioItem key={url.url} value={url.url}>
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
