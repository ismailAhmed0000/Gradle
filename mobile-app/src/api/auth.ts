import { apiClient } from './client';

export type Role = 'teacher' | 'admin' | 'student';

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

// GRADLE_AUTH_REDIRECT is the app's registered deep link scheme (see
// ios/MobileApp/Info.plist and android's AndroidManifest.xml) — Google never
// sees this directly, only the backend's own HTTPS callback does; the
// backend redirects here as its very last step once it has issued a Gradle
// JWT for the signed-in student.
export const GOOGLE_AUTH_REDIRECT = 'gradleapp://auth-callback';

export async function fetchGoogleSignInUrl(): Promise<string> {
  const response = await apiClient.get<{ url: string }>(
    '/api/integrations/google/student/auth-url',
    { params: { redirect_uri: GOOGLE_AUTH_REDIRECT } },
  );
  return response.data.url;
}

export async function fetchCurrentUser(token: string): Promise<User> {
  const response = await apiClient.get<User>('/api/auth/me', {
    headers: { Authorization: `Bearer ${token}` },
  });
  return response.data;
}
