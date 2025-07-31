// src/services/chatApi.ts

import type { Chat, ChatListResponse, Message } from "../types/chat";

// Мокаем данные для демонстрации
const mockUsers = [
  {
    id: 1,
    firstName: "Алексей",
    lastName: "Петров",
    username: "alexey_p",
    photoUrl: "https://i.pravatar.cc/150?img=1",
  },
  {
    id: 2,
    firstName: "Мария",
    lastName: "Иванова",
    username: "maria_i",
    photoUrl: "https://i.pravatar.cc/150?img=2",
  },
  {
    id: 3,
    firstName: "React Developers",
    photoUrl: "https://i.pravatar.cc/150?img=3",
  },
];

const mockMessages: Message[] = [
  {
    id: 1,
    text: "Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?Привет! Как дела?",
    userId: 1,
    timestamp: new Date(Date.now() - 1000 * 60 * 30),
    isRead: true,
    messageType: "text",
  },
  {
    id: 2,
    text: "ОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтОтлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?Отлично! А у тебя как?",
    userId: 2,
    timestamp: new Date(Date.now() - 1000 * 60 * 15),
    isRead: false,
    messageType: "text",
  },
  {
    id: 3,
    text: "Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?Кто-нибудь разбирается в TypeScript?",
    userId: 3,
    timestamp: new Date(Date.now() - 1000 * 60 * 60),
    isRead: true,
    messageType: "text",
  },
];

const mockChats: Chat[] = [
  {
    id: 1,
    title: "Алексей Петров",
    type: "private",
    participants: [mockUsers[0]],
    lastMessage: mockMessages[0],
    unreadCount: 2,
    photoUrl: mockUsers[0].photoUrl,
    isOnline: true,
    username: "swallow",
  },
  {
    id: 2,
    title: "Мария Иванова",
    type: "private",
    participants: [mockUsers[1]],
    lastMessage: mockMessages[1],
    unreadCount: 0,
    photoUrl: mockUsers[1].photoUrl,
    isOnline: false,
    lastSeen: new Date(Date.now() - 1000 * 60 * 5),
  },
  {
    id: 3,
    title: "React Developers",
    type: "group",
    participants: mockUsers,
    lastMessage: mockMessages[2],
    unreadCount: 5,
    photoUrl: mockUsers[2].photoUrl,
    username: "fisting",
  },
];

// Симуляция API вызовов
export const chatApi = {
  async getChats(offset = 0, limit = 20): Promise<ChatListResponse> {
    // Симуляция задержки сети
    await new Promise((resolve) => setTimeout(resolve, 500));

    const start = offset;
    const end = offset + limit;
    const chats = mockChats.slice(start, end);

    return {
      chats,
      total: mockChats.length,
      hasMore: end < mockChats.length,
    };
  },

  async getChatById(id: number): Promise<Chat | null> {
    await new Promise((resolve) => setTimeout(resolve, 300));
    return mockChats.find((chat) => chat.id === id) || null;
  },

  async getChatMessages(
    chatId: number,
    offset = 0,
    limit = 50,
  ): Promise<Message[]> {
    await new Promise((resolve) => setTimeout(resolve, 400));
    // В реальном приложении здесь был бы запрос к API
    return mockMessages.filter((msg) => Math.random() > 0.3);
  },

  async sendMessage(chatId: number, text: string): Promise<Message> {
    await new Promise((resolve) => setTimeout(resolve, 200));

    const newMessage: Message = {
      id: Date.now(),
      text,
      userId: 0, // Текущий пользователь
      timestamp: new Date(),
      isRead: false,
      messageType: "text",
    };

    return newMessage;
  },
};
