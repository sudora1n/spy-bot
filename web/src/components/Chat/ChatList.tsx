import { FC, useCallback } from "react";
import {
  List,
  Spinner,
  Placeholder,
  Divider,
} from "@telegram-apps/telegram-ui";
import { ChatItem } from "./ChatItem";
import { useChats } from "../../hooks/useChats";
import type { Chat } from "../../types/chat";
import Lottie from "lottie-react";

import internalError from "@/assets/lottie/internal_error.json";

interface ChatListProps {
  onChatClick: (chat: Chat) => void;
}

export const ChatList: FC<ChatListProps> = ({ onChatClick }) => {
  const { chats, loading, error, hasMore, loadMoreChats, refreshChats } =
    useChats();

  const handleScroll = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;

      if (
        scrollHeight - scrollTop <= clientHeight * 1.5 &&
        hasMore &&
        !loading
      ) {
        loadMoreChats();
      }
    },
    [hasMore, loading, loadMoreChats],
  );

  if (error) {
    return (
      <Placeholder
        header="Ошибка загрузки"
        description={error}
        action={
          <button
            onClick={refreshChats}
            className="px-4 py-2 bg-telegram-button text-telegram-button rounded-lg hover:scale-105 transition-transform"
          >
            Попробовать снова
          </button>
        }
      >
        <Lottie
          animationData={internalError}
          loop={true}
          className="block w-36 h-36" // 144px 144px <3
        />
      </Placeholder>
    );
  }

  return (
    <div className="h-full overflow-y-auto" onScroll={handleScroll}>
      <List>
        <div className="">
          {chats.map((chat, index) => (
            <div key={chat.id}>
              <ChatItem chat={chat} onClick={onChatClick} />
              {index !== chats.length - 1 && (
                <Divider className="border-telegram-separator" />
              )}
            </div>
          ))}
        </div>

        {loading && (
          <div className="flex justify-center p-4">
            <Spinner size="s" />
          </div>
        )}

        {!hasMore && chats.length > 0 && (
          <div className="text-center text-telegram-hint p-4 text-sm">
            Больше чатов нет
          </div>
        )}
      </List>
    </div>
  );
};
