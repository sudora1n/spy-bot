import { FC, useRef, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Avatar, Spinner, Placeholder, Cell } from "@telegram-apps/telegram-ui";
import { Page } from "@/components/Page.tsx";
import { MessageBubble } from "../components/Chat/MessageBubble";
import { useChat } from "../hooks/useChats";
import Lottie from "lottie-react";

import internalError from "@/assets/lottie/internal_error.json";

export const ChatDetailPage: FC = () => {
  const { chatId } = useParams<{ chatId: string }>();
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const { chat, messages, loading, error } = useChat(
    chatId ? parseInt(chatId) : 0,
  );

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  if (loading) {
    return (
      <Page>
        <div className="flex items-center justify-center p-8">
          <Spinner size="m" />
        </div>
      </Page>
    );
  }

  if (error || !chat) {
    return (
      <Page>
        <Placeholder header="Ошибка" description={error || "Чат не найден"}>
          <Lottie
            animationData={internalError}
            loop={true}
            className="block w-36 h-36" // 144px 144px <3
          />
        </Placeholder>
      </Page>
    );
  }

  return (
    <Page>
      <div className="sticky top-0 z-10 bg-telegram-header">
        <Cell
          before={
            <div className="relative">
              <Avatar
                size={40}
                src={chat.photoUrl}
                fallbackIcon={chat.title.charAt(0)}
              />
            </div>
          }
          subtitle={chat.username ? `@${chat.username}` : undefined}
        >
          {chat.title}
        </Cell>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-2">
        {messages.length === 0 ? (
          <div className="flex items-center justify-center h-full">
            <p className="text-gray-500">Нет сообщений</p>
          </div>
        ) : (
          messages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              isOwn={message.userId === 0}
            />
          ))
        )}
        <div ref={messagesEndRef} />
      </div>
    </Page>
  );
};
