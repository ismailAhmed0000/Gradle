import { useCallback, useEffect } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  Text,
  View,
} from 'react-native';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useAssignments } from '../context/AssignmentsContext';
import { Assignment } from '../api/assignments';
import { TasksStackParamList } from '../navigation/types';

type Props = NativeStackScreenProps<TasksStackParamList, 'AssignmentsList'>;

export function AssignmentsListScreen({ navigation }: Props) {
  const { list, listStatus, listError, loadAssignments } = useAssignments();

  useEffect(() => {
    loadAssignments();
  }, [loadAssignments]);

  const renderItem = useCallback(
    ({ item }: { item: Assignment }) => (
      <Pressable
        onPress={() =>
          navigation.navigate('AssignmentDetail', { assignmentId: item.id })
        }
        className="border-b border-gray-100 px-6 py-4"
      >
        <Text className="text-base font-semibold text-gray-900">
          {item.title}
        </Text>
        <Text className="mt-1 text-xs text-gray-500">
          {new Date(item.created_at).toLocaleDateString()}
        </Text>
      </Pressable>
    ),
    [navigation],
  );

  if (listStatus === 'loading' && list.length === 0) {
    return (
      <View className="flex-1 items-center justify-center bg-white">
        <ActivityIndicator size="large" color="#2f6690" />
      </View>
    );
  }

  if (listStatus === 'error') {
    return (
      <View className="flex-1 items-center justify-center bg-white px-6">
        <Text className="text-center text-sm text-red-600">{listError}</Text>
      </View>
    );
  }

  return (
    <FlatList
      className="flex-1 bg-white"
      data={list}
      keyExtractor={item => item.id}
      renderItem={renderItem}
      refreshControl={
        <RefreshControl
          refreshing={listStatus === 'loading'}
          onRefresh={() => loadAssignments(true)}
        />
      }
      ListEmptyComponent={
        <View className="items-center px-6 py-12">
          <Text className="text-sm text-gray-500">No assignments yet.</Text>
        </View>
      }
    />
  );
}
