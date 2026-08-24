import { useCallback, useEffect } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useAssignments } from '../context/AssignmentsContext';
import {
  ASSIGNMENT_STATUS_COLORS,
  ASSIGNMENT_STATUS_LABELS,
  Assignment,
} from '../api/assignments';
import { TasksStackParamList } from '../navigation/types';

const ACCENT_COLOR = '#2f6690';

type Props = NativeStackScreenProps<TasksStackParamList, 'AssignmentsList'>;

export function AssignmentsListScreen({ navigation }: Props) {
  const insets = useSafeAreaInsets();
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
        className="mb-3 flex-row items-center justify-between rounded-3xl bg-gray-50 p-5"
      >
        <View className="flex-1 pr-4">
          <Text className="text-base font-semibold text-gray-900">
            {item.title}
          </Text>
          <Text className="mt-1 text-xs text-gray-500">
            {item.due_date
              ? `Due ${new Date(item.due_date).toLocaleDateString()}`
              : 'No due date'}
          </Text>
        </View>
        <Text
          className="text-xs font-semibold"
          style={{ color: ASSIGNMENT_STATUS_COLORS[item.status] }}
        >
          {ASSIGNMENT_STATUS_LABELS[item.status]}
        </Text>
      </Pressable>
    ),
    [navigation],
  );

  if (listStatus === 'loading' && list.length === 0) {
    return (
      <View className="flex-1 items-center justify-center bg-white">
        <ActivityIndicator size="large" color={ACCENT_COLOR} />
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
    <View className="flex-1 bg-white">
      <View
        className="px-6 pb-2"
        style={{ paddingTop: Math.max(insets.top, 16) + 12 }}
      >
        <Text className="text-center text-3xl font-bold text-gray-900">
          Assignments
        </Text>
      </View>
      <FlatList
        className="flex-1"
        contentContainerStyle={styles.listContent}
        data={list}
        keyExtractor={item => item.id}
        renderItem={renderItem}
        refreshControl={
          <RefreshControl
            refreshing={listStatus === 'loading'}
            onRefresh={() => loadAssignments(true)}
            tintColor={ACCENT_COLOR}
          />
        }
        ListEmptyComponent={
          <View className="items-center px-6 py-12">
            <Text className="text-sm text-gray-500">No assignments yet.</Text>
          </View>
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  listContent: {
    paddingHorizontal: 24,
    paddingVertical: 16,
  },
});
