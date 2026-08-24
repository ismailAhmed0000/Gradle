import { useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useAuth } from '../context/AuthContext';

const ACCENT_COLOR = '#2f6690';

type Props = {
  onSwitchToLogin: () => void;
};

export function RegisterScreen({ onSwitchToLogin }: Props) {
  const { register } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit() {
    if (!email || !password) {
      setError('Enter your email and password.');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match.');
      return;
    }
    setError(null);
    setIsSubmitting(true);
    try {
      await register(email.trim(), password);
    } catch {
      setError('Could not create account. Try a different email.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <View className="flex-1 justify-center bg-white px-6">
      <Text className="mb-8 text-center text-3xl font-bold text-gray-900">
        Create an account
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
          className="mb-4 rounded-xl border border-gray-200 bg-white px-4 py-3 text-base"
        />

        <Text className="mb-1 text-xs font-semibold tracking-wide text-gray-400">
          CONFIRM PASSWORD
        </Text>
        <TextInput
          value={confirmPassword}
          onChangeText={setConfirmPassword}
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
          <Text className="text-base font-semibold text-white">Sign up</Text>
        )}
      </Pressable>

      <Pressable onPress={onSwitchToLogin} className="mt-6 items-center">
        <Text
          className="text-sm font-semibold"
          style={{ color: ACCENT_COLOR }}
        >
          Already have an account? Log in
        </Text>
      </Pressable>
    </View>
  );
}
