import { useEffect } from 'react';
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from 'react-native';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import Pdf from 'react-native-pdf';
import { useAssignments } from '../context/AssignmentsContext';
import { TasksStackParamList } from '../navigation/types';
import { CameraIcon } from '../components/BottomNavBar';

type Props = NativeStackScreenProps<TasksStackParamList, 'AssignmentDetail'>;

export function AssignmentDetailScreen({ route }: Props) {
  const { assignmentId } = route.params;
  const {
    detailsById,
    detailStatusById,
    detailErrorById,
    loadAssignmentDetail,
  } = useAssignments();

  const detail = detailsById[assignmentId];
  const status = detailStatusById[assignmentId] ?? 'idle';
  const error = detailErrorById[assignmentId];

  useEffect(() => {
    loadAssignmentDetail(assignmentId);
  }, [assignmentId, loadAssignmentDetail]);

  function handleStartScan() {}

  if (status === 'loading' && !detail) {
    return (
      <View className="flex-1 items-center justify-center bg-white">
        <ActivityIndicator size="large" color="#2f6690" />
      </View>
    );
  }

  if (status === 'error' && !detail) {
    return (
      <View className="flex-1 items-center justify-center bg-white px-6">
        <Text className="text-center text-sm text-red-600">{error}</Text>
      </View>
    );
  }

  if (!detail) {
    return null;
  }

  const questionPaper = detail.assignment_files[0];

  return (
    <View className="flex-1 bg-white">
      <View className="px-6 pt-6">
        <Text className="text-2xl font-bold text-gray-900">{detail.title}</Text>
        <Text className="mt-1 text-xs text-gray-500">
          {new Date(detail.created_at).toLocaleDateString()}
        </Text>
      </View>

      <View className="mt-4 flex-1 bg-gray-100">
        {questionPaper?.download_url ? (
          <Pdf
            source={{ uri: questionPaper.download_url, cache: true }}
            trustAllCerts={false}
            style={styles.pdf}
          />
        ) : (
          <View className="flex-1 items-center justify-center">
            <Text className="text-sm text-gray-500">
              No question paper uploaded yet.
            </Text>
          </View>
        )}
      </View>

      <View className="items-center border-t border-gray-100 py-6">
        <Pressable onPress={handleStartScan} style={scanButtonStyle}>
          <CameraIcon color="#ffffff" />
        </Pressable>
        <Text className="mt-2 text-xs text-gray-500">Start scanning</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  pdf: {
    flex: 1,
    width: '100%',
  },
});

const scanButtonStyle = {
  width: 64,
  height: 64,
  borderRadius: 32,
  backgroundColor: '#2f6690',
  alignItems: 'center' as const,
  justifyContent: 'center' as const,
};
