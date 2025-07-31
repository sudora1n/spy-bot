import { Placeholder, AppRoot } from "@telegram-apps/telegram-ui";
import {
  retrieveLaunchParams,
  isColorDark,
  isRGB,
} from "@telegram-apps/sdk-react";
import { useMemo } from "react";
import Lottie from "lottie-react";

import internalError from "@/assets/lottie/internal_error.json";

export function EnvUnsupported() {
  const [platform, isDark] = useMemo(() => {
    try {
      const lp = retrieveLaunchParams();
      const { bg_color: bgColor } = lp.tgWebAppThemeParams;
      return [
        lp.tgWebAppPlatform,
        bgColor && isRGB(bgColor) ? isColorDark(bgColor) : false,
      ];
    } catch {
      return ["android", false];
    }
  }, []);

  return (
    <AppRoot
      appearance={isDark ? "dark" : "light"}
      platform={["macos", "ios"].includes(platform) ? "ios" : "base"}
    >
      <Placeholder
        header="Oops"
        description="You are using too old Telegram client to run this application"
      >
        <Lottie
          animationData={internalError}
          loop={true}
          className="block w-36 h-36" // 144px 144px <3
        />
      </Placeholder>
    </AppRoot>
  );
}
