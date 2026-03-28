import type { DeviceRow, EventRow } from '@/types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api';

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function fetchApi<T>(
  path: string,
  adminKey: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      ...options?.headers,
      Authorization: `Bearer ${adminKey}`,
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new ApiError('Unauthorized - check your admin key', 401);
    }
    const text = await response.text().catch(() => 'Unknown error');
    throw new ApiError(text || `HTTP ${response.status}`, response.status);
  }

  return response.json();
}

function normalizeArrayResponse<T>(value: T[] | null): T[] {
  return Array.isArray(value) ? value : [];
}

export async function fetchDevices(adminKey: string): Promise<DeviceRow[]> {
  const response = await fetchApi<DeviceRow[] | null>('/devices', adminKey);
  return normalizeArrayResponse(response);
}

export async function fetchLatestEvent(
  adminKey: string,
  deviceId: string
): Promise<EventRow> {
  return fetchApi<EventRow>(
    `/events/latest?device_id=${encodeURIComponent(deviceId)}`,
    adminKey
  );
}

export async function fetchEvents(
  adminKey: string,
  deviceId: string,
  limit: number = 50
): Promise<EventRow[]> {
  const response = await fetchApi<EventRow[] | null>(
    `/events?device_id=${encodeURIComponent(deviceId)}&limit=${limit}`,
    adminKey
  );
  return normalizeArrayResponse(response);
}

export async function checkHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE_URL}/healthz`);
    return response.ok;
  } catch {
    return false;
  }
}
