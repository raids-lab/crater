// ignore-i18n-script
import {
  BadgeCheck,
  BookOpenIcon,
  ChevronsUpDown,
  Globe,
  LogOut,
  Moon,
  Sparkles,
  Sun,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { useAtomValue, useSetAtom } from "jotai";
import {
  globalAccount,
  globalHideUsername,
  globalLastView,
  globalUserInfo,
  useResetStore,
} from "@/utils/store";
import { Link, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTheme } from "@/utils/theme";
import { toast } from "sonner";
import { Role } from "@/services/api/auth";
import useIsAdmin from "@/hooks/useAdmin";
import { configUrlWebsiteBaseAtom } from "@/utils/store/config";
import { UserAvatar } from "../custom/UserDetail/UserAvatar";
import { getUserPseudonym } from "@/utils/pseudonym";
import { useTranslation } from "react-i18next";

export function NavUser() {
  const website = useAtomValue(configUrlWebsiteBaseAtom);
  const { isMobile } = useSidebar();
  const queryClient = useQueryClient();
  const { resetAll } = useResetStore();
  const user = useAtomValue(globalUserInfo);
  const account = useAtomValue(globalAccount);
  const navigate = useNavigate();
  const { theme, setTheme } = useTheme();
  const setLastView = useSetAtom(globalLastView);
  const isAdminView = useIsAdmin();
  const hideUsername = useAtomValue(globalHideUsername);
  const { t, i18n } = useTranslation();

  const displayName = hideUsername
    ? getUserPseudonym(user.name)
    : user.nickname || user.name;

  const changeLanguage = (lng: "zh" | "en" | "ja" | "ko") => {
    i18n.changeLanguage(lng);
  };

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <UserAvatar user={user} />
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">{displayName}</span>
                <span className="truncate text-xs">{user.email}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <UserAvatar user={user} />
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">{displayName}</span>
                  <span className="truncate text-xs">{user.email}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            {account.rolePlatform === Role.Admin && (
              <>
                <DropdownMenuGroup>
                  <DropdownMenuItem
                    onClick={() => {
                      if (isAdminView) {
                        setLastView("portal");
                        navigate("/portal");
                        toast.success(t("navUser.switchToUserView"));
                      } else {
                        setLastView("admin");
                        navigate("/admin");
                        toast.success(t("navUser.switchToAdminView"));
                      }
                    }}
                  >
                    <Sparkles />
                    {t("navUser.switchTo") +
                      (isAdminView
                        ? t("navUser.normalUser")
                        : t("navUser.admin"))}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
              </>
            )}
            <DropdownMenuGroup>
              <DropdownMenuItem asChild>
                <Link to="/portal/setting/user">
                  <BadgeCheck />
                  {t("navUser.personalPage")}
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => window.open(website)}>
                <BookOpenIcon />
                {t("navUser.platformDocs")}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              >
                {theme === "light" ? <Moon /> : <Sun />}
                {theme === "light"
                  ? t("navUser.darkMode")
                  : t("navUser.lightMode")}
              </DropdownMenuItem>
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  <Globe className="text-muted-foreground mr-2 h-4 w-4" />
                  {t("navUser.language")}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent>
                  <DropdownMenuRadioGroup value={i18n.language}>
                    <DropdownMenuRadioItem
                      value="zh"
                      onClick={() => changeLanguage("zh")}
                    >
                      {t("navUser.chinese")}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem
                      value="en"
                      onClick={() => changeLanguage("en")}
                    >
                      {t("navUser.english")}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem
                      value="ja"
                      onClick={() => changeLanguage("ja")}
                    >
                      {t("navUser.japanese")}
                    </DropdownMenuRadioItem>
                    <DropdownMenuRadioItem
                      value="ko"
                      onClick={() => changeLanguage("ko")}
                    >
                      {t("navUser.korean")}
                    </DropdownMenuRadioItem>
                  </DropdownMenuRadioGroup>
                </DropdownMenuSubContent>
              </DropdownMenuSub>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="focus:bg-destructive focus:text-destructive-foreground"
              onClick={() => {
                setLastView(isAdminView ? "admin" : "portal");
                queryClient.clear();
                resetAll();
                toast.success(t("navUser.loggedOut"));
              }}
            >
              <LogOut className="dark:text-destructive-foreground" />
              {t("navUser.logout")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
