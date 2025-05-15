// i18n-processed-v1.1.0
import { useTranslation } from "react-i18next";
import { IAccount } from "@/services/api/account";
import { useState } from "react";
import { AccountTable } from "./account-table";
import { AccountForm } from "./account-form-component";
import SandwichSheet from "@/components/sheet/SandwichSheet";

export const Component = () => {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [currentAccount, setCurrentAccount] = useState<IAccount | null>(null);

  return (
    <>
      <AccountTable
        setIsOpen={setIsOpen}
        setCurrentAccount={setCurrentAccount}
      />
      <SandwichSheet
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title={
          currentAccount
            ? t("accountForm.editTitle")
            : t("accountForm.createTitle")
        }
        description={t("accountForm.description")}
        className="sm:max-w-4xl"
      >
        <AccountForm onOpenChange={setIsOpen} account={currentAccount} />
      </SandwichSheet>
    </>
  );
};
