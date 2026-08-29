import { useState } from 'react';
import {
  ActivityIndicator,
  Linking,
  Pressable,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useAuth } from '../context/AuthContext';
import { fetchGoogleSignInUrl } from '../api/auth';

const ACCENT_COLOR = '#2f6690';

type Props = {
  onSwitchToRegister: () => void;
};

export function LoginScreen({ onSwitchToRegister }: Props) {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isOpeningGoogle, setIsOpeningGoogle] = useState(false);

  async function handleGoogleSignIn() {
    setError(null);
    setIsOpeningGoogle(true);
    try {
      const url = await fetchGoogleSignInUrl();
      await Linking.openURL(url);
    } catch {
      setError('Could not start Google sign-in. Please try again.');
    } finally {
      setIsOpeningGoogle(false);
    }
  }

  async function handleSubmit() {
    if (!email || !password) {
      setError('Enter your email and password.');
      return;
    }
    setError(null);
    setIsSubmitting(true);
    try {
      await login(email.trim(), password);
    } catch {
      setError('Invalid email or password.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <View className="flex-1 justify-center bg-white px-6">
      <Text className="mb-8 text-center text-3xl font-bold text-gray-900">
        Welcome back
      </Text>

      <View className="rounded-3xl bg-gray-50 p-6">
        <Text className="mb-1 text-xs font-semibold tracking-wide text-gray-400">
          EMAIL
        </Text>
        <TextInput
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="email-address"
          placeholder="you@example.com"
          className="mb-4 rounded-xl border border-gray-200 bg-white px-4 py-3 text-base"
        />

        <Text className="mb-1 text-xs font-semibold tracking-wide text-gray-400">
          PASSWORD
        </Text>
        <TextInput
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          placeholder="••••••••"
          className="rounded-xl border border-gray-200 bg-white px-4 py-3 text-base"
        />
      </View>

      {error && (
        <Text className="mt-3 text-center text-sm text-red-600">{error}</Text>
      )}

      <Pressable
        onPress={handleSubmit}
        disabled={isSubmitting}
        style={{ backgroundColor: ACCENT_COLOR }}
        className="mt-6 items-center rounded-xl py-3.5 disabled:opacity-60"
      >
        {isSubmitting ? (
          <ActivityIndicator color="#ffffff" />
        ) : (
          <Text className="text-base font-semibold text-white">Log in</Text>
        )}
      </Pressable>

      <Pressable onPress={onSwitchToRegister} className="mt-6 items-center">
        <Text
          className="text-sm font-semibold"
          style={{ color: ACCENT_COLOR }}
        >
          Don't have an account? Sign up
        </Text>
      </Pressable>

      <View className="mt-6 flex-row items-center">
        <View className="h-px flex-1 bg-gray-200" />
        <Text className="mx-3 text-xs font-semibold text-gray-400">
          STUDENTS
        </Text>
        <View className="h-px flex-1 bg-gray-200" />
      </View>

      <Pressable
        onPress={handleGoogleSignIn}
        disabled={isOpeningGoogle}
        className="mt-4 items-center rounded-xl border border-gray-200 py-3.5 disabled:opacity-60"
      >
        {isOpeningGoogle ? (
          <ActivityIndicator color={ACCENT_COLOR} />
        ) : (
          <Text className="text-base font-semibold text-gray-900">
            Continue with Google
          </Text>
        )}
      </Pressable>
    </View>
  );
}
