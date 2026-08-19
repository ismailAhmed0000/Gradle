import { Text, View } from 'react-native';

export function TasksScreen() {
  return (
    <View className="flex-1 bg-white px-6 pt-8">
      <Text className="text-2xl font-bold text-gray-900">Tasks</Text>
      <Text className="mt-2 text-sm text-gray-500">
        Submissions you've scanned will show up here.
      </Text>
    </View>
  );
}
