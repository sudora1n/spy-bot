import { useState, useEffect, useCallback } from "react";
import { chatApi } from "../services/chatApi";
import type { Chat, Message } from "../types/chat";

export const useChats = () => {
  const [chats, setChats] = useState<Chat[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(true);

  const loadChats = useCallback(
    async (offset = 0) => {
      if (loading) return;

      setLoading(true);
      setError(null);

      try {
        const response = await chatApi.getChats(offset);

        if (offset === 0) {
          setChats(response.chats);
        } else {
          setChats((prev) => [...prev, ...response.chats]);
        }

        setHasMore(response.hasMore);
      } catch (err) {
        setError("Ошибка загрузки чатов");
        console.error("Error loading chats:", err);
      } finally {
        setLoading(false);
      }
    },
    [loading],
  );

  useEffect(() => {
    loadChats();
  }, []);

  const loadMoreChats = useCallback(() => {
    if (hasMore && !loading) {
      loadChats(chats.length);
    }
  }, [chats.length, hasMore, loading, loadChats]);

  const refreshChats = useCallback(() => {
    setChats([]);
    setHasMore(true);
    loadChats(0);
  }, [loadChats]);

  return {
    chats,
    loading,
    error,
    hasMore,
    loadMoreChats,
    refreshChats,
  };
};

export const useChat = (chatId: number) => {
  const [chat, setChat] = useState<Chat | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadChat = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const [chatData, messagesData] = await Promise.all([
        chatApi.getChatById(chatId),
        chatApi.getChatMessages(chatId),
      ]);

      setChat(chatData);
      setMessages(messagesData);
    } catch (err) {
      setError("Ошибка загрузки чата");
      console.error("Error loading chat:", err);
    } finally {
      setLoading(false);
    }
  }, [chatId]);

  useEffect(() => {
    if (chatId) {
      loadChat();
    }
  }, [chatId, loadChat]);

  return {
    chat,
    messages,
    loading,
    error,
    refreshChat: loadChat,
  };
};
