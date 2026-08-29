import { createContext, useMemo, useEffect, useState, useContext } from 'react';
import {
  fetchCurrentUser,
  loginRequest,
  registerRequest,
  User,
} from '../api/auth';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { TOKEN_STORAGE_KEY } from '../api/tokenStorage';

type AuthContextValue = {
  user: User | null;
  token: string | null;
  isBootstrapping: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  loginWithToken: (token: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isBootstrapping, setIsBootstrapping] = useState(true);

  useEffect(() => {
    bootstrap();
  }, []);

  async function bootstrap() {
    try {
      const storedToken = await AsyncStorage.getItem(TOKEN_STORAGE_KEY);
      if (!storedToken) {
        return;
      }
      const currentUser = await fetchCurrentUser(storedToken);
      setToken(storedToken);
      setUser(currentUser);
    } catch {
      await AsyncStorage.removeItem(TOKEN_STORAGE_KEY);
    } finally {
      setIsBootstrapping(false);
    }
  }

  async function login(email: string, password: string) {
    const response = await loginRequest(email, password);
    await AsyncStorage.setItem(TOKEN_STORAGE_KEY, response.token);
    setToken(response.token);
    setUser(response.user);
  }

  async function register(email: string, password: string) {
    const response = await registerRequest(email, password);
    await AsyncStorage.setItem(TOKEN_STORAGE_KEY, response.token);
    setToken(response.token);
    setUser(response.user);
  }

  // Used after the Google sign-in deep link hands back a Gradle JWT the
  // backend already issued — no credentials to post, just adopt the token.
  async function loginWithToken(newToken: string) {
    const currentUser = await fetchCurrentUser(newToken);
    await AsyncStorage.setItem(TOKEN_STORAGE_KEY, newToken);
    setToken(newToken);
    setUser(currentUser);
  }

  async function logout() {
    await AsyncStorage.removeItem(TOKEN_STORAGE_KEY);
    setToken(null);
    setUser(null);
  }

  const value = useMemo(
    () => ({ user, token, isBootstrapping, login, register, loginWithToken, logout }),
    [user, token, isBootstrapping],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
