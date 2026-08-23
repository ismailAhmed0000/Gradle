import { useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Modal,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useIsFocused } from '@react-navigation/native';
import Pdf from 'react-native-pdf';
import { useAssignments } from '../context/AssignmentsContext';
import { useSubmissions } from '../context/SubmissionsContext';
import { TasksStackParamList } from '../navigation/types';
import { CameraIcon } from '../components/BottomNavBar';
import {
  createSubmission,
  Submission,
  SubmissionSummary,
  uploadSubmissionPage,
} from '../api/submissions';
import { capturePage } from '../utils/scan';
import { fetchPdfAsDataUri } from '../utils/pdf';

const STATUS_LABELS: Record<SubmissionSummary['status'], string> = {
  pending: 'Pending',
  processing: 'Processing',
  composited: 'Graded',
  failed: 'Failed',
};

type Props = NativeStackScreenProps<TasksStackParamList, 'AssignmentDetail'>;

export function AssignmentDetailScreen({ route, navigation }: Props) {
  const { assignmentId } = route.params;
  // Avoid keeping the (fairly heavy) native PDF view alive underneath
  // SubmissionDetail once the user navigates away from this screen.
  const isFocused = useIsFocused();
  const {
    detailsById,
    detailStatusById,
    detailErrorById,
    loadAssignmentDetail,
  } = useAssignments();

  const detail = detailsById[assignmentId];
  const status = detailStatusById[assignmentId] ?? 'idle';
  const error = detailErrorById[assignmentId];

  const { submissionsByAssignmentId, loadSubmissions } = useSubmissions();
  const submissions = submissionsByAssignmentId[assignmentId] ?? [];

  const [showNameModal, setShowNameModal] = useState(false);
  const [studentName, setStudentName] = useState('');
  const [creatingSubmission, setCreatingSubmission] = useState(false);
  const [submission, setSubmission] = useState<Submission | null>(null);
  const [capturing, setCapturing] = useState(false);
  const [nextPageNumber, setNextPageNumber] = useState(1);
  const [pagesCaptured, setPagesCaptured] = useState(0);
  const [pagesUploaded, setPagesUploaded] = useState(0);
  const [pageErrors, setPageErrors] = useState(0);

  useEffect(() => {
    loadAssignmentDetail(assignmentId);
  }, [assignmentId, loadAssignmentDetail]);

  useEffect(() => {
    loadSubmissions(assignmentId);
  }, [assignmentId, loadSubmissions]);

  const questionPaperURL = detail?.assignment_files[0]?.download_url;
  const [pdfDataUri, setPdfDataUri] = useState<string | null>(null);
  const [pdfLoadFailed, setPdfLoadFailed] = useState(false);

  useEffect(() => {
    if (!questionPaperURL) {
      return;
    }
    let cancelled = false;
    setPdfDataUri(null);
    setPdfLoadFailed(false);
    // react-native-pdf's remote-URL loader reliably stalls on anything but
    // the very first PDF fetched in the app session (a bug in the shared
    // react-native-blob-util streaming path) — fetch as base64 ourselves
    // instead, which goes through a different, working code path.
    fetchPdfAsDataUri(questionPaperURL)
      .then(dataUri => {
        if (!cancelled) setPdfDataUri(dataUri);
      })
      .catch(() => {
        if (!cancelled) setPdfLoadFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [questionPaperURL]);

  // Keep this object's identity stable across re-renders (capturing,
  // pagesCaptured, etc. all tick during a scan session) — a fresh literal
  // every render makes react-native-pdf treat it as a new source and
  // restart the load, so it never finishes.
  const pdfSource = useMemo(
    () => (pdfDataUri ? { uri: pdfDataUri } : null),
    [pdfDataUri],
  );

  function handleStartScan() {
    setStudentName('');
    setShowNameModal(true);
  }

  async function handleConfirmStudent() {
    const name = studentName.trim();
    if (!name) {
      return;
    }
    setCreatingSubmission(true);
    try {
      const created = await createSubmission(assignmentId, name);
      setSubmission(created);
      setNextPageNumber(1);
      setPagesCaptured(0);
      setPagesUploaded(0);
      setPageErrors(0);
      setShowNameModal(false);
      handleCapture(created);
    } catch {
      Alert.alert('Could not start scan', 'Please try again.');
    } finally {
      setCreatingSubmission(false);
    }
  }

  async function handleCapture(activeSubmission: Submission | null = submission) {
    if (!activeSubmission || capturing) {
      return;
    }
    setCapturing(true);
    const photo = await capturePage();
    setCapturing(false);
    if (!photo) {
      return;
    }

    const pageNumber = nextPageNumber;
    setNextPageNumber(n => n + 1);
    setPagesCaptured(n => n + 1);

    // Fire the upload in the background so the camera can reopen for the
    // next page immediately instead of waiting on the network round trip.
    uploadSubmissionPage(activeSubmission.id, pageNumber, photo)
      .then(() => setPagesUploaded(n => n + 1))
      .catch(() => setPageErrors(n => n + 1));
  }

  function handleFinishScanning() {
    setSubmission(null);
    setNextPageNumber(1);
    setPagesCaptured(0);
    setPagesUploaded(0);
    setPageErrors(0);
    loadSubmissions(assignmentId, true);
  }

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

  const pagesUploading = pagesCaptured - pagesUploaded - pageErrors;

  return (
    <View className="flex-1 bg-white">
      <View className="px-6 pt-6">
        <Text className="text-2xl font-bold text-gray-900">{detail.title}</Text>
        <Text className="mt-1 text-xs text-gray-500">
          {new Date(detail.created_at).toLocaleDateString()}
        </Text>
      </View>

      <View className="mt-4 flex-1 bg-gray-100">
        {pdfSource && isFocused ? (
          <Pdf source={pdfSource} trustAllCerts={false} style={styles.pdf} />
        ) : (
          <View className="flex-1 items-center justify-center px-6">
            {!questionPaperURL ? (
              <Text className="text-sm text-gray-500">
                No question paper uploaded yet.
              </Text>
            ) : pdfLoadFailed ? (
              <Text className="text-center text-sm text-red-600">
                Failed to load the question paper.
              </Text>
            ) : (
              <ActivityIndicator color="#2f6690" />
            )}
          </View>
        )}
      </View>

      {submissions.length > 0 && (
        <View className="border-t border-gray-100" style={{ maxHeight: 176 }}>
          <Text className="px-6 pt-3 text-xs font-semibold tracking-wide text-gray-400">
            SUBMISSIONS
          </Text>
          <FlatList
            data={submissions}
            keyExtractor={item => item.id}
            renderItem={({ item }) => (
              <Pressable
                onPress={() =>
                  navigation.navigate('SubmissionDetail', {
                    submissionId: item.id,
                    studentName: item.student_name,
                  })
                }
                className="flex-row items-center justify-between px-6 py-3"
              >
                <View>
                  <Text className="text-sm font-semibold text-gray-900">
                    {item.student_name}
                  </Text>
                  <Text className="mt-0.5 text-xs text-gray-500">
                    {item.page_count} page{item.page_count === 1 ? '' : 's'}
                    {item.answer_regions_total > 0
                      ? ` · ${item.answer_regions_done}/${item.answer_regions_total} extracted`
                      : ''}
                  </Text>
                </View>
                <Text className="text-xs font-medium text-blue-600">
                  {STATUS_LABELS[item.status]}
                </Text>
              </Pressable>
            )}
          />
        </View>
      )}

      <View className="items-center border-t border-gray-100 py-6">
        {!submission ? (
          <>
            <Pressable
              onPress={handleStartScan}
              style={scanButtonStyle}
              disabled={creatingSubmission}
            >
              {creatingSubmission ? (
                <ActivityIndicator color="#ffffff" />
              ) : (
                <CameraIcon color="#ffffff" />
              )}
            </Pressable>
            <Text className="mt-2 text-xs text-gray-500">Start scanning</Text>
          </>
        ) : (
          <View className="w-full px-6">
            <Text className="text-center text-sm font-semibold text-gray-900">
              {submission.student_name}
            </Text>
            <Text className="mt-1 text-center text-xs text-gray-500">
              {pagesCaptured} page{pagesCaptured === 1 ? '' : 's'} captured
              {pagesUploading > 0 ? ` · uploading ${pagesUploading}` : ''}
              {pageErrors > 0 ? ` · ${pageErrors} failed` : ''}
            </Text>
            <View className="mt-4 items-center">
              <Pressable
                onPress={() => handleCapture()}
                style={scanButtonStyle}
                disabled={capturing}
              >
                {capturing ? (
                  <ActivityIndicator color="#ffffff" />
                ) : (
                  <CameraIcon color="#ffffff" />
                )}
              </Pressable>
              <Text className="mt-2 text-xs text-gray-500">Scan next page</Text>
            </View>
            <Pressable onPress={handleFinishScanning} className="mt-4 items-center">
              <Text className="text-sm font-semibold text-blue-600">
                Done scanning
              </Text>
            </Pressable>
          </View>
        )}
      </View>

      <Modal
        visible={showNameModal}
        transparent
        animationType="fade"
        onRequestClose={() => setShowNameModal(false)}
      >
        <View className="flex-1 items-center justify-center bg-black/40 px-8">
          <View className="w-full rounded-2xl bg-white p-6">
            <Text className="text-lg font-bold text-gray-900">
              Student name
            </Text>
            <TextInput
              value={studentName}
              onChangeText={setStudentName}
              placeholder="e.g. Jane Doe"
              autoFocus
              className="mt-4 rounded-xl border border-gray-200 px-4 py-3 text-base"
            />
            <View className="mt-5 flex-row justify-end gap-4">
              <Pressable onPress={() => setShowNameModal(false)}>
                <Text className="text-sm text-gray-500">Cancel</Text>
              </Pressable>
              <Pressable
                onPress={handleConfirmStudent}
                disabled={!studentName.trim() || creatingSubmission}
              >
                <Text className="text-sm font-semibold text-blue-600">
                  {creatingSubmission ? 'Starting…' : 'Start'}
                </Text>
              </Pressable>
            </View>
          </View>
        </View>
      </Modal>
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
