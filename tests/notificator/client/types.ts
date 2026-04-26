export interface NotificationItem {
  id: string;
  title: string;
  type: string;
  text: string;
  sentAt: string;
  readAt: string | null;
}

export interface ListNotificationsResponse {
  notifications: NotificationItem[];
}

export interface NotificationSetting {
  type: string;
  enabled: boolean;
}

export interface GetNotificationSettingsResponse {
  settings: NotificationSetting[];
}

export interface ToggleNotificationResponse {
  enabled: boolean;
}

export interface ListNotificationsParams {
  page?: number;
  pageSize?: number;
}

export interface CreateTestNotificationRequest {
  user_id: string;
  title: string;
  type: string;
  text?: string;
}

export interface CreateTestNotificationResponse {
  id: string;
}
