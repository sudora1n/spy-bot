import { Avatar } from "@telegram-apps/telegram-ui";
import { Page } from "@/components/Page.tsx";
import { retrieveLaunchParams } from "@telegram-apps/sdk-react";

export function IndexPage() {
  const launchParams = retrieveLaunchParams();

  return (
    <Page back={false}>
      <div className="flex items-center justify-center p-8">
        <div className="max-w-3xs">
          <div className="flex items-center bg-telegram-secondary border-telegram-separator py-1 px-2.5 w-fit rounded-xl space-x-2.5">
            <Avatar
              size={24}
              src={launchParams.tgWebAppData?.user?.photo_url}
              alt="avatar"
              className="rounded-[50%]"
            />
            <span className="text-telegram-hint text-sm font-medium">
              {launchParams.tgWebAppData?.user?.first_name}
            </span>
          </div>
        </div>
      </div>
    </Page>
  );
}
