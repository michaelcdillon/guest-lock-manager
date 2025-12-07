/**
 * API Client Service
 * 
 * Provides type-safe access to the REST API endpoints.
 */

const API_BASE = '/api';

/** Standard API error response */
interface ApiError {
  error: string;
  message: string;
  details?: unknown;
}

/** Health check response */
export interface HealthResponse {
  status: string;
  ha_connected: boolean;
  db_connected: boolean;
}

/** System status response */
export interface StatusResponse {
  ha_connected: boolean;
  ha_version?: string;
  zwave_js_ui_available: boolean;
  zigbee2mqtt_available: boolean;
  calendars_count: number;
  locks_count: number;
  active_guest_pins: number;
  active_static_pins: number;
  next_sync_at?: string;
  pending_operations: number;
}

/** Calendar subscription */
export interface Calendar {
  id: string;
  name: string;
  url: string;
  sync_interval_min: number;
  last_sync_at?: string;
  sync_status: 'pending' | 'syncing' | 'success' | 'error';
  sync_error?: string;
  enabled: boolean;
}

/** Managed lock */
export interface Lock {
  id: string;
  entity_id: string;
  name: string;
  protocol: string;
  total_slots: number;
  guest_slots: number;
  static_slots: number;
  online: boolean;
  state: string;
  battery_level?: number;
  last_seen_at?: string;
  direct_integration?: string;
}

/** Guest PIN */
export interface GuestPin {
  id: string;
  calendar_id: string;
  event_uid: string;
  event_summary?: string;
  pin_code: string;
  generation_method: string;
  valid_from: string;
  valid_until: string;
  status: 'pending' | 'active' | 'expired' | 'conflict';
}

/** Static PIN */
export interface StaticPin {
  id: string;
  name: string;
  pin_code: string;
  enabled: boolean;
  always_active: boolean;
  schedules?: PinSchedule[];
}

/** PIN schedule (day/time restriction) */
export interface PinSchedule {
  id: string;
  day_of_week: number;
  start_time: string;
  end_time: string;
}

/** Settings */
export interface Settings {
  default_sync_interval_min: string;
  min_pin_length: string;
  max_pin_length: string;
  checkin_time: string;
  checkout_time: string;
  battery_efficient_mode: string;
  batch_window_seconds: string;
}

/**
 * Make an API request with error handling.
 */
async function request<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const url = `${API_BASE}${path}`;
  const options: RequestInit = {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(url, options);

  if (!response.ok) {
    const error: ApiError = await response.json().catch(() => ({
      error: 'unknown_error',
      message: `HTTP ${response.status}: ${response.statusText}`,
    }));
    throw new Error(error.message);
  }

  // Handle empty responses
  const text = await response.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text) as T;
}

/**
 * API client with methods for all endpoints.
 */
export const api = {
  // Health & Status
  getHealth: () => request<HealthResponse>('GET', '/health'),
  getStatus: () => request<StatusResponse>('GET', '/status'),

  // Calendars
  getCalendars: () => request<Calendar[]>('GET', '/calendars'),
  getCalendar: (id: string) => request<Calendar>('GET', `/calendars/${id}`),
  createCalendar: (data: Partial<Calendar>) =>
    request<Calendar>('POST', '/calendars', data),
  updateCalendar: (id: string, data: Partial<Calendar>) =>
    request<Calendar>('PUT', `/calendars/${id}`, data),
  deleteCalendar: (id: string) => request<void>('DELETE', `/calendars/${id}`),
  syncCalendar: (id: string) =>
    request<{ status: string }>('POST', `/calendars/${id}/sync`),
  getCalendarLocks: (id: string) => request<string[]>('GET', `/calendars/${id}/locks`),
  updateCalendarLocks: (id: string, lockIds: string[]) =>
    request<void>('PUT', `/calendars/${id}/locks`, { lock_ids: lockIds }),

  // Locks
  getLocks: () => request<Lock[]>('GET', '/locks'),
  getLock: (id: string) => request<Lock>('GET', `/locks/${id}`),
  discoverLocks: () => request<Lock[]>('POST', '/locks/discover'),
  updateLock: (id: string, data: Partial<Lock>) =>
    request<Lock>('PUT', `/locks/${id}`, data),
  deleteLock: (id: string) => request<void>('DELETE', `/locks/${id}`),

  // Guest PINs
  getGuestPins: (params?: { calendar_id?: string; status?: string }) => {
    const query = new URLSearchParams();
    if (params?.calendar_id) query.set('calendar_id', params.calendar_id);
    if (params?.status) query.set('status', params.status);
    const queryStr = query.toString();
    return request<GuestPin[]>('GET', `/guest-pins${queryStr ? `?${queryStr}` : ''}`);
  },
  getGuestPin: (id: string) => request<GuestPin>('GET', `/guest-pins/${id}`),
  updateGuestPin: (id: string, data: { custom_pin?: string }) =>
    request<GuestPin>('PATCH', `/guest-pins/${id}`, data),
  regenerateGuestPin: (id: string) =>
    request<GuestPin>('POST', `/guest-pins/${id}/regenerate`),

  // Static PINs
  getStaticPins: () => request<StaticPin[]>('GET', '/static-pins'),
  getStaticPin: (id: string) => request<StaticPin>('GET', `/static-pins/${id}`),
  createStaticPin: (data: Partial<StaticPin>) =>
    request<StaticPin>('POST', '/static-pins', data),
  updateStaticPin: (id: string, data: Partial<StaticPin>) =>
    request<StaticPin>('PUT', `/static-pins/${id}`, data),
  deleteStaticPin: (id: string) => request<void>('DELETE', `/static-pins/${id}`),

  // Settings
  getSettings: () => request<Settings>('GET', '/settings'),
  updateSettings: (data: Partial<Settings>) =>
    request<Settings>('PUT', '/settings', data),
};

