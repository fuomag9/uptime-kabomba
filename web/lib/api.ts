// API client for backend communication
// In production (Docker), API requests are proxied through Next.js to backend
// In development, you can set NEXT_PUBLIC_API_URL to http://localhost:8080

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';

export interface LoginRequest {
  username: string;
  password: string;
  token?: string; // 2FA token
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface User {
  id: number;
  username: string;
  email?: string;
  provider?: string;
  active: boolean;
  created_at: string;
}

export interface OAuthConfig {
  enabled: boolean;
  issuer?: string;
}

export interface OAuthCallbackResponse {
  action: 'login' | 'link_required' | 'register' | 'error';
  token?: string;
  user?: User;
  linking_token?: string;
  email?: string;
  message?: string;
}

export interface LinkAccountRequest {
  linking_token: string;
  password: string;
}

export interface ApiError {
  message: string;
  status: number;
}

class ApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;

    // Load token from localStorage on client side
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('token');
    }
  }

  setToken(token: string | null) {
    this.token = token;
    if (typeof window !== 'undefined') {
      if (token) {
        localStorage.setItem('token', token);
      } else {
        localStorage.removeItem('token');
      }
    }
  }

  getToken(): string | null {
    return this.token;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      // Clear token on authentication errors
      if (response.status === 401) {
        this.setToken(null);
      }

      const error: ApiError = {
        message: await response.text(),
        status: response.status,
      };
      throw error;
    }

    // Handle empty responses
    const text = await response.text();
    if (!text || text === 'null') {
      return null as T;
    }

    return JSON.parse(text);
  }

  // Auth endpoints
  async login(data: LoginRequest): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    this.setToken(response.token);
    return response;
  }

  async setup(data: LoginRequest): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/api/auth/setup', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    this.setToken(response.token);
    return response;
  }

  async logout(): Promise<void> {
    await this.request('/api/auth/logout', {
      method: 'POST',
    });
    this.setToken(null);
  }

  async getCurrentUser(): Promise<User> {
    return this.request<User>('/api/user/me');
  }

  // OAuth endpoints
  async getOAuthConfig(): Promise<OAuthConfig> {
    return this.request<OAuthConfig>('/api/auth/oauth/config');
  }

  async linkOAuthAccount(data: LinkAccountRequest): Promise<LoginResponse> {
    const response = await this.request<LoginResponse>('/api/auth/oauth/link', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    this.setToken(response.token);
    return response;
  }

  async getSetupStatus(): Promise<{ setupComplete: boolean }> {
    return this.request<{ setupComplete: boolean }>('/api/auth/status');
  }

  // Monitor endpoints
  async getMonitors(): Promise<MonitorWithStatus[]> {
    const result = await this.request<MonitorWithStatus[] | null>('/api/monitors');
    return result || [];
  }

  async getMonitor(id: number): Promise<Monitor> {
    return this.request<Monitor>(`/api/monitors/${id}`);
  }

  async createMonitor(data: CreateMonitorRequest): Promise<Monitor> {
    return this.request<Monitor>('/api/monitors', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateMonitor(id: number, data: UpdateMonitorRequest): Promise<Monitor> {
    return this.request<Monitor>(`/api/monitors/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteMonitor(id: number): Promise<void> {
    return this.request<void>(`/api/monitors/${id}`, {
      method: 'DELETE',
    });
  }

  async getMonitorNotifications(monitorId: number): Promise<Notification[]> {
    const result = await this.request<Notification[] | null>(
      `/api/monitors/${monitorId}/notifications`
    );
    return result || [];
  }

  async updateMonitorNotifications(
    monitorId: number,
    notificationIds: number[]
  ): Promise<Notification[]> {
    return this.request<Notification[]>(
      `/api/monitors/${monitorId}/notifications`,
      {
        method: 'PUT',
        body: JSON.stringify({ notification_ids: notificationIds }),
      }
    );
  }

  async getHeartbeats(
    monitorId: number,
    options?: {
      limit?: number;
      period?: '24h' | '7d' | '30d' | '90d';
      startTime?: string;
      endTime?: string;
    }
  ): Promise<Heartbeat[]> {
    const params = new URLSearchParams();
    if (options?.limit) {
      params.append('limit', options.limit.toString());
    }
    if (options?.period) {
      params.append('period', options.period);
    }
    if (options?.startTime) {
      params.append('start_time', options.startTime);
    }
    if (options?.endTime) {
      params.append('end_time', options.endTime);
    }
    const queryString = params.toString();
    const url = `/api/monitors/${monitorId}/heartbeats${queryString ? `?${queryString}` : ''}`;
    const result = await this.request<Heartbeat[] | null>(url);
    return result || [];
  }

  async getUserSettings(): Promise<UserSettings> {
    return this.request<UserSettings>('/api/settings');
  }

  async updateUserSettings(settings: {
    heartbeat_retention_days: number;
    hourly_stat_retention_days: number;
    daily_stat_retention_days: number;
  }): Promise<UserSettings> {
    return this.request<UserSettings>('/api/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    });
  }

  // Notification endpoints
  async getNotifications(): Promise<Notification[]> {
    const result = await this.request<Notification[] | null>('/api/notifications');
    return result || [];
  }

  async getNotification(id: number): Promise<Notification> {
    return this.request<Notification>(`/api/notifications/${id}`);
  }

  async createNotification(data: CreateNotificationRequest): Promise<Notification> {
    return this.request<Notification>('/api/notifications', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateNotification(id: number, data: UpdateNotificationRequest): Promise<Notification> {
    return this.request<Notification>(`/api/notifications/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteNotification(id: number): Promise<void> {
    return this.request<void>(`/api/notifications/${id}`, {
      method: 'DELETE',
    });
  }

  async testNotification(id: number): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/api/notifications/${id}/test`, {
      method: 'POST',
    });
  }

  async getNotificationProviders(): Promise<NotificationProvider[]> {
    const result = await this.request<NotificationProvider[] | null>('/api/notifications/providers');
    return result || [];
  }

  // Status Page endpoints
  async getStatusPages(): Promise<StatusPage[]> {
    const result = await this.request<StatusPage[] | null>('/api/status-pages');
    return result || [];
  }

  async getStatusPage(id: number): Promise<StatusPageWithMonitors> {
    return this.request<StatusPageWithMonitors>(`/api/status-pages/${id}`);
  }

  async createStatusPage(data: CreateStatusPageRequest): Promise<StatusPage> {
    return this.request<StatusPage>('/api/status-pages', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateStatusPage(id: number, data: UpdateStatusPageRequest): Promise<StatusPage> {
    return this.request<StatusPage>(`/api/status-pages/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteStatusPage(id: number): Promise<void> {
    return this.request<void>(`/api/status-pages/${id}`, {
      method: 'DELETE',
    });
  }

  async getPublicStatusPage(slug: string, password?: string): Promise<PublicStatusPage> {
    const headers: Record<string, string> = {};
    if (password) {
      headers['X-Status-Page-Password'] = password;
    }
    return this.request<PublicStatusPage>(`/status/${slug}`, { headers });
  }

  async getIncidents(statusPageId: number): Promise<Incident[]> {
    const result = await this.request<Incident[] | null>(`/api/status-pages/${statusPageId}/incidents`);
    return result || [];
  }

  async createIncident(statusPageId: number, data: CreateIncidentRequest): Promise<Incident> {
    return this.request<Incident>(`/api/status-pages/${statusPageId}/incidents`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async deleteIncident(statusPageId: number, incidentId: number): Promise<void> {
    return this.request<void>(`/api/status-pages/${statusPageId}/incidents/${incidentId}`, {
      method: 'DELETE',
    });
  }
}

// Monitor types
export interface Monitor {
  id: number;
  user_id: number;
  name: string;
  type: string;
  url: string;
  interval: number;
  timeout: number;
  resend_interval: number;
  ip_version: string;
  active: boolean;
  config: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface CreateMonitorRequest {
  name: string;
  type: string;
  url: string;
  interval?: number;
  timeout?: number;
  resend_interval?: number;
  ip_version?: string;
  config?: Record<string, any>;
}

export interface UpdateMonitorRequest extends CreateMonitorRequest {
  active?: boolean;
}

export interface MonitorNotificationLink {
  notification_ids: number[];
}

export interface Heartbeat {
  id: number;
  monitor_id: number;
  status: number; // 0=down, 1=up, 2=pending, 3=maintenance
  ping: number;
  important: boolean;
  message: string;
  time: string;
}

export interface UserSettings {
  id: number;
  user_id: number;
  heartbeat_retention_days: number;
  hourly_stat_retention_days: number;
  daily_stat_retention_days: number;
  created_at: string;
  updated_at: string;
}

// Notification types
export interface Notification {
  id: number;
  user_id: number;
  name: string;
  type: string;
  config: string; // JSON string from backend
  is_default: boolean;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateNotificationRequest {
  name: string;
  type: string;
  config: Record<string, any>;
  is_default?: boolean;
  active?: boolean;
}

export interface UpdateNotificationRequest extends CreateNotificationRequest {}

export interface NotificationProvider {
  name: string;
  label: string;
}

// Status Page types
export interface StatusPage {
  id: number;
  user_id: number;
  slug: string;
  title: string;
  description: string;
  published: boolean;
  show_powered_by: boolean;
  theme: string;
  custom_css: string;
  created_at: string;
  updated_at: string;
}

export interface StatusPageWithMonitors {
  id: number;
  user_id: number;
  slug: string;
  title: string;
  description: string;
  published: boolean;
  show_powered_by: boolean;
  theme: string;
  custom_css: string;
  created_at: string;
  updated_at: string;
  monitors: Monitor[];
}

export interface CreateStatusPageRequest {
  slug: string;
  title: string;
  description?: string;
  published?: boolean;
  show_powered_by?: boolean;
  theme?: string;
  custom_css?: string;
  password?: string;
  monitor_ids?: number[];
}

export interface UpdateStatusPageRequest extends CreateStatusPageRequest {}

export interface Incident {
  id: number;
  status_page_id: number;
  title: string;
  content: string;
  style: string; // info, warning, danger, success
  pin: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateIncidentRequest {
  title: string;
  content: string;
  style?: string;
  pin?: boolean;
}

export interface MonitorWithStatus extends Monitor {
  last_heartbeat?: Heartbeat;
}

export interface PublicStatusPage {
  page: StatusPage;
  monitors: MonitorWithStatus[];
  incidents: Incident[];
}

export const apiClient = new ApiClient(API_BASE_URL);
