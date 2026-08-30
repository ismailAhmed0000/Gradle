import { ActivityIndicator, Text, View } from 'react-native';

type Props = {
  message?: string;
};

export function SplashScreen({ message = 'Loading…' }: Props) {
  return (
    <View className="flex-1 items-center justify-center bg-white">
      <ActivityIndicator size="large" color="#4f46e5" />
      <Text className="mt-4 text-base text-gray-500">{message}</Text>
    </View>
  );
}
