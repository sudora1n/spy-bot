import { FC } from "react";
import { useNavigate } from "react-router-dom";
import { List } from "@telegram-apps/telegram-ui";
import { Page } from "@/components/Page.tsx";
import { ChatList } from "../components/Chat/ChatList";
import type { Chat } from "../types/chat";

export const ChatsPage: FC = () => {
  const navigate = useNavigate();

  const handleChatClick = (chat: Chat) => {
    navigate(`/chats/${chat.id}`);
  };

  return (
    <Page back={false}>
      <List>
        <ChatList onChatClick={handleChatClick} />
      </List>
    </Page>
  );
};
