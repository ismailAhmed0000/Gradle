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
      <Text className="mb-8 text-center text-2xl font-bold text-gray-900">
        Create an account
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
        className="mb-4 rounded-lg border border-gray-300 px-4 py-3 text-base"
      />

      <Text className="mb-1 text-sm font-medium text-gray-700">
        Confirm password
      </Text>
      <TextInput
        value={confirmPassword}
        onChangeText={setConfirmPassword}
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
          <Text className="text-base font-semibold text-white">Sign up</Text>
        )}
      </Pressable>

      <Pressable onPress={onSwitchToLogin} className="mt-6 items-center">
        <Text className="text-sm text-indigo-600">
          Already have an account? Log in
        </Text>
      </Pressable>
    </View>
  );
}
