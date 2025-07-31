import { FC, useState } from "react";
import { useLocation } from "react-router-dom";
import { Link } from "@/components/Link";
import { House, MessageCircleMore } from "lucide-react";

interface BottomBarProps {
  className?: string;
}

export const TelegramBottomBar: FC<BottomBarProps> = ({ className = "" }) => {
  const location = useLocation();
  const currentPath = location.pathname;
  const [animatingTab, setAnimatingTab] = useState<string | null>(null);

  const handleTabClick = (tabId: string) => {
    setAnimatingTab(tabId);
    setTimeout(() => setAnimatingTab(null), 150);
  };

  const tabs = [
    {
      id: "home",
      text: "Home",
      path: "/",
      icon: (isActive: boolean) => (
        <House
          className={`w-5 h-5 transition-colors duration-200 ${
            isActive ? "text-telegram-button" : "text-telegram-hint"
          }`}
        />
      ),
    },
    {
      id: "chats",
      text: "Chats",
      path: "/chats",
      icon: (isActive: boolean) => (
        <MessageCircleMore
          className={`w-5 h-5 transition-colors duration-200 ${
            isActive ? "text-telegram-button" : "text-telegram-hint"
          }`}
        />
      ),
    },
  ];

  return (
    <div className={`fixed bottom-0 left-0 right-0 z-50 ${className}`}>
      <div className="bg-telegram-bottom-bar px-4 py-3">
        <div className="flex justify-center">
          <div className="flex rounded-full p-1 shadow-lg space-x-2.5">
            {tabs.map((tab) => {
              const isActive = (() => {
                if (tab.id === "chats") return currentPath.startsWith("/chats");
                return currentPath === tab.path;
              })();

              return (
                <Link key={tab.id} to={tab.path}>
                  <div
                    onClick={() => handleTabClick(tab.id)}
                    className={`
                      flex flex-col items-center justify-center
                      px-5 py-2 rounded-full cursor-pointer min-w-[70px]
                      ${isActive ? "active bg-telegram-button" : "hover:bg-telegram-secondary"}
                      ${animatingTab === tab.id ? "tab-select-animation" : ""}
                    `}
                  >
                    <div
                      className={`telegram-tab-icon mb-1 ${
                        animatingTab === tab.id ? "scale-95" : ""
                      }`}
                    >
                      {tab.icon(isActive)}
                    </div>
                    <span
                      className={`text-xs font-medium transition-colors duration-200 ${
                        isActive ? "text-telegram-button" : "text-telegram-hint"
                      }`}
                    >
                      {tab.text}
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
};
