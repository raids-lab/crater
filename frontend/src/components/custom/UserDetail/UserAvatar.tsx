// i18n-processed-v1.1.0
// Modified code
import { useTranslation } from "react-i18next";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import Identicon from "@polkadot/react-identicon";
import { stringToSS58 } from "@/utils/ss58";
import { cn } from "@/lib/utils";

interface UserAvatarProps {
  user: {
    avatar?: string;
    name: string;
  };
  size?: number;
  className?: string;
  identiconTheme?: "beachball" | "polkadot" | "substrate";
}

export function UserAvatar({
  user,
  size = 32,
  className,
  identiconTheme = "beachball",
}: UserAvatarProps) {
  const { t } = useTranslation();

  return (
    <Avatar className={cn("size-8", className)}>
      <AvatarImage src={user.avatar} alt={t("userAvatar.alt")} />
      <AvatarFallback>
        <Identicon
          value={stringToSS58(user.name)}
          size={size}
          theme={identiconTheme}
          className="cursor-default!"
        />
      </AvatarFallback>
    </Avatar>
  );
}
