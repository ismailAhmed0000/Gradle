import { useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useAuth } from '../context/AuthContext';

type Props = {
  onSwitchToRegister: () => void;
};

export function LoginScreen({ onSwitchToRegister }: Props) {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

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
      <Text className="mb-8 text-center text-2xl font-bold text-gray-900">
        Welcome back
      </Text>

      <Text className="mb-1 text-sm font-medium text-gray-700">Email</Text>
      <TextInput
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoCorrect={false}
        keyboardType="email-address"
        placeholder="you@example.com"
        className="mb-4 rounded-lg border border-gray-300 px-4 py-3 text-base"
      />

      <Text className="mb-1 text-sm font-medium text-gray-700">Password</Text>
      <TextInput
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        placeholder="••••••••"
        className="mb-2 rounded-lg border border-gray-300 px-4 py-3 text-base"
      />

      {error && <Text className="mb-2 text-sm text-red-600">{error}</Text>}

      <Pressable
        onPress={handleSubmit}
        disabled={isSubmitting}
        className="mt-4 items-center rounded-lg bg-indigo-600 py-3 disabled:opacity-60"
      >
        {isSubmitting ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text className="text-base font-semibold text-white">Log in</Text>
        )}
      </Pressable>

      <Pressable onPress={onSwitchToRegister} className="mt-6 items-center">
        <Text className="text-sm text-indigo-600">
          Don't have an account? Sign up
        </Text>
      </Pressable>
    </View>
  );
}
