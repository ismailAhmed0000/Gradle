import { Pressable, Text, View } from 'react-native';
import { useAuth } from '../context/AuthContext';

export function HomeScreen() {
  const { user, logout } = useAuth();

  return (
    <View className="flex-1 bg-white px-6 pt-8">
      <Text className="text-2xl font-bold text-gray-900">
        Welcome{user ? `, ${user.email}` : ''}
      </Text>
      {user && (
        <Text className="mt-1 text-sm text-gray-500">Role: {user.role}</Text>
      )}

      <Pressable
        onPress={logout}
        className="mt-8 items-center rounded-lg bg-gray-900 py-3"
      >
        <Text className="text-base font-semibold text-white">Log out</Text>
      </Pressable>
    </View>
  );
}
