import { IUserAttributes } from "@/services/api/admin/user";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";
import Identicon from "@polkadot/react-identicon";
import { stringToSS58 } from "@/utils/ss58";
import { Link } from "react-router-dom";

const TooltipUser = ({ attributes }: { attributes: IUserAttributes }) => {
  return (
    <TooltipProvider delayDuration={50}>
      <Tooltip>
        <TooltipTrigger>
          <span className="truncate text-sm font-normal">
            {attributes.nickname ?? attributes.name}
          </span>
        </TooltipTrigger>
        <TooltipContent className="bg-background text-foreground border">
          <Link
            className="m-0 p-0 font-normal"
            to={`/portal/user/${attributes.name}`}
          >
            <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
              <Avatar className="h-8 w-8">
                <AvatarImage src={attributes.avatar} alt="Avatar preview" />
                <AvatarFallback>
                  <Identicon
                    value={stringToSS58(attributes.name)}
                    size={32}
                    theme="beachball"
                    className="cursor-default!"
                  />
                </AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">
                  {attributes.nickname}
                </span>
                <span className="truncate text-xs">@{attributes.name}</span>
              </div>
            </div>
          </Link>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export default TooltipUser;
