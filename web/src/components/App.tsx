import { useMemo } from "react";
import { Navigate, Route, Routes, HashRouter } from "react-router-dom";
import {
  retrieveLaunchParams,
  useSignal,
  isMiniAppDark,
} from "@telegram-apps/sdk-react";
import { AppRoot } from "@telegram-apps/telegram-ui";

import { routes } from "@/navigation/routes.tsx";
import { TelegramBottomBar } from "./Navigaton/BottomBar";

export function App() {
  const lp = useMemo(() => retrieveLaunchParams(), []);
  const isDark = useSignal(isMiniAppDark);

  return (
    <AppRoot
      appearance={isDark ? "dark" : "light"}
      platform={["macos", "ios"].includes(lp.tgWebAppPlatform) ? "ios" : "base"}
    >
      <HashRouter>
        {/* Основной контент с отступом снизу для bottom bar */}
        <div className="min-h-screen pb-16">
          <Routes>
            {routes.map((route) => (
              <Route key={route.path} {...route} />
            ))}
            <Route path="*" element={<Navigate to="/" />} />
          </Routes>
        </div>

        {/* Bottom Bar в стиле Telegram */}
        <TelegramBottomBar />
      </HashRouter>
    </AppRoot>
  );
}
