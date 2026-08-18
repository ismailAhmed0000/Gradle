import { apiClient } from './client';

export type Role = 'teacher' | 'admin';

export type User = {
  id: string;
  email: string;
  role: Role;
  created_at: string;
};

export type AuthResponse = {
  token: string;
  user: User;
};

export async function loginRequest(
  email: string,
  password: string,
): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/api/auth/login', {
    email,
    password,
  });
  return response.data;
}

export async function registerRequest(
  email: string,
  password: string,
): Promise<AuthResponse> {
  const response = await apiClient.post<AuthResponse>('/api/auth/register', {
    email,
    password,
  });
  return response.data;
}

export async function fetchCurrentUser(token: string): Promise<User> {
  const response = await apiClient.get<User>('/api/auth/me', {
    headers: { Authorization: `Bearer ${token}` },
  });
  return response.data;
}
