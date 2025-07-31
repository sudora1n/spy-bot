import type { ComponentType } from "react";

import { ChatsPage } from "@/pages/ChatsPage";
import { ChatDetailPage } from "@/pages/ChatDetailPage";
import { IndexPage } from "@/pages/IndexPage";

interface Route {
  path: string;
  Component: ComponentType;
  title?: string;
  icon?: ComponentType;
}

export const routes: Route[] = [
  { path: "/", Component: IndexPage },

  { path: "/chats", Component: ChatsPage, title: "Чаты" },
  { path: "/chats/:chatId", Component: ChatDetailPage, title: "Чат" },
];
