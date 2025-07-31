import { FC } from "react";
import type { Message } from "../../types/chat";

interface MessageBubbleProps {
  message: Message;
  isOwn: boolean;
}

export const MessageBubble: FC<MessageBubbleProps> = ({ message, isOwn }) => {
  const formatTime = (date: Date) => {
    return date.toLocaleTimeString("ru-RU", {
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <div className={`flex mb-1 ${isOwn ? "justify-end" : "justify-start"}`}>
      <div
        className={`
          max-w-xs lg:max-w-md rounded-2xl border-0 shadow-sm px-3 py-2
          ${
            isOwn
              ? "bg-telegram-button text-telegram-button rounded-br-md"
              : "bg-telegram-secondary text-telegram rounded-bl-md"
          }
        `}
      >
        <div className="flex items-end gap-2">
          <div className="text-sm whitespace-pre-line wrap-anywhere">
            {message.text}
          </div>
          <div className="text-xs opacity-60">
            {formatTime(message.timestamp)}
          </div>
        </div>
      </div>
    </div>
  );
};
