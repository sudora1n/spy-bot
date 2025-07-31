import { FC } from "react";
import { Avatar, Cell } from "@telegram-apps/telegram-ui";
import type { Chat } from "../../types/chat";

interface ChatItemProps {
  chat: Chat;
  onClick: (chat: Chat) => void;
}

export const ChatItem: FC<ChatItemProps> = ({ chat, onClick }) => {
  const formatTime = (date: Date) => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / (1000 * 60));
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (minutes < 1) return "сейчас";
    if (minutes < 60) return `${minutes}м`;
    if (hours < 24) return `${hours}ч`;
    if (days < 7) return `${days}д`;

    return date.toLocaleDateString("ru-RU", {
      day: "2-digit",
      month: "2-digit",
    });
  };

  return (
    <Cell
      before={
        <div className="relative">
          <Avatar
            size={48}
            src={chat.photoUrl}
            fallbackIcon={chat.title.charAt(0)}
          />
        </div>
      }
      after={
        <div className="flex flex-col items-end">
          <span className="text-xs text-telegram-hint mb-1">
            {chat.lastMessage && formatTime(chat.lastMessage.timestamp)}
          </span>
        </div>
      }
      subtitle={
        <div className="space-y-1">
          <div className="flex items-center text-sm text-telegram-hint truncate">
            {chat.lastMessage?.text || "Нет сообщений"}
          </div>
        </div>
      }
      onClick={() => onClick(chat)}
    >
      {chat.title}
    </Cell>
  );
};
