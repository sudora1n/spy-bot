export interface User {
  id: number;
  firstName: string;
  lastName?: string;
  username?: string;
  photoUrl?: string;
}

export interface Message {
  id: number;
  text: string;
  userId: number;
  timestamp: Date;
  isRead: boolean;
  messageType: "text" | "photo" | "video" | "document";
}

export interface Chat {
  id: number;
  title: string;
  username?: string;
  type: "private" | "group" | "channel";
  participants: User[];
  lastMessage?: Message;
  unreadCount: number;
  photoUrl?: string;
  isOnline?: boolean;
  lastSeen?: Date;
}

export interface ChatListResponse {
  chats: Chat[];
  total: number;
  hasMore: boolean;
}
