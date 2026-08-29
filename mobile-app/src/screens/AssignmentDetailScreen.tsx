import { useEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useIsFocused } from '@react-navigation/native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Svg, { Path } from 'react-native-svg';
import Pdf from 'react-native-pdf';
import { useAssignments } from '../context/AssignmentsContext';
import { useAuth } from '../context/AuthContext';
import { useSubmissions } from '../context/SubmissionsContext';
import { TasksStackParamList } from '../navigation/types';
import { CameraIcon } from '../components/BottomNavBar';
import {
  createSubmission,
  Submission,
  SubmissionSummary,
  uploadSubmissionPage,
} from '../api/submissions';
import {
  ASSIGNMENT_STATUS_COLORS,
  ASSIGNMENT_STATUS_LABELS,
} from '../api/assignments';
import { capturePage } from '../utils/scan';
import { fetchPdfAsDataUri } from '../utils/pdf';

const ACCENT_COLOR = '#2f6690';

function BackIcon() {
  return (
    <Svg width={24} height={24} viewBox="0 0 24 24" fill="none">
      <Path
        d="M19 12H5M5 12l7-7M5 12l7 7"
        stroke="#111827"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Svg>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <View className="flex-row items-center justify-between py-2.5">
      <Text className="text-sm text-gray-500">{label}</Text>
      <Text className="text-sm font-semibold text-gray-900">{value}</Text>
    </View>
  );
}

const STATUS_LABELS: Record<SubmissionSummary['status'], string> = {
  pending: 'Pending',
  processing: 'Processing',
  composited: 'Graded',
  failed: 'Failed',
};

type Tab = 'question' | 'answer';

type Props = NativeStackScreenProps<TasksStackParamList, 'AssignmentDetail'>;

export function AssignmentDetailScreen({ route, navigation }: Props) {
  const { assignmentId } = route.params;
  const { user } = useAuth();
  const isStudent = user?.role === 'student';
  // Avoid keeping the (fairly heavy) native PDF view alive underneath
  // SubmissionDetail once the user navigates away from this screen.
  const isFocused = useIsFocused();
  const insets = useSafeAreaInsets();
  const [activeTab, setActiveTab] = useState<Tab>('question');
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
  // The download_url is a presigned S3 link that expires after ~15 minutes;
  // assignment detail is otherwise cached indefinitely, so a screen left
  // open (or revisited) past that window would hold a dead URL. Retry once
  // by refreshing the assignment detail (which mints a new URL) before
  // surfacing an error.
  const pdfRetriedRef = useRef(false);

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
        if (cancelled) {
          return;
        }
        if (!pdfRetriedRef.current) {
          pdfRetriedRef.current = true;
          loadAssignmentDetail(assignmentId, true);
        } else {
          setPdfLoadFailed(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [questionPaperURL, assignmentId, loadAssignmentDetail]);

  // Keep this object's identity stable across re-renders (capturing,
  // pagesCaptured, etc. all tick during a scan session) — a fresh literal
  // every render makes react-native-pdf treat it as a new source and
  // restart the load, so it never finishes.
  const pdfSource = useMemo(
    () => (pdfDataUri ? { uri: pdfDataUri } : null),
    [pdfDataUri],
  );

  function handleStartScan() {
    // A student submits as themselves — the backend derives their name from
    // their own linked roster record, so there's nothing to prompt for.
    if (isStudent) {
      startSubmission('');
      return;
    }
    setStudentName('');
    setShowNameModal(true);
  }

  async function startSubmission(name: string) {
    setCreatingSubmission(true);
    try {
      const created = await createSubmission(assignmentId, name);
      const startPage = created.page_count + 1;
      setSubmission(created);
      setNextPageNumber(startPage);
      setPagesCaptured(created.page_count);
      setPagesUploaded(created.page_count);
      setPageErrors(0);
      setShowNameModal(false);
      handleCapture(created, startPage);
    } catch {
      Alert.alert('Could not start scan', 'Please try again.');
    } finally {
      setCreatingSubmission(false);
    }
  }

  async function handleConfirmStudent() {
    const name = studentName.trim();
    if (!name) {
      return;
    }
    await startSubmission(name);
  }

  async function handleCapture(
    activeSubmission: Submission | null = submission,
    // handleConfirmStudent calls this synchronously right after
    // setNextPageNumber, so the `nextPageNumber` in this closure would
    // otherwise still be the pre-update value — pass the real one explicitly
    // instead of falling back to state in that case.
    pageNumberOverride?: number,
  ) {
    if (!activeSubmission || capturing) {
      return;
    }
    setCapturing(true);
    const photo = await capturePage();
    setCapturing(false);
    if (!photo) {
      return;
    }

    const pageNumber = pageNumberOverride ?? nextPageNumber;
    setNextPageNumber(pageNumber + 1);
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
      <View
        className="px-6"
        style={{ paddingTop: Math.max(insets.top, 16) + 12 }}
      >
        <Pressable onPress={() => navigation.goBack()} hitSlop={8}>
          <BackIcon />
        </Pressable>
        <View className="mt-3 flex-row items-start justify-between">
          <Text className="flex-1 pr-4 text-2xl font-bold text-gray-900">
            {detail.title}
          </Text>
          <Text
            className="text-xs font-semibold"
            style={{ color: ASSIGNMENT_STATUS_COLORS[detail.status] }}
          >
            {ASSIGNMENT_STATUS_LABELS[detail.status]}
          </Text>
        </View>
      </View>

      <View className="mt-4 flex-row justify-center gap-10 border-b border-gray-100 px-6">
        <Pressable onPress={() => setActiveTab('question')}>
          <Text
            className="pb-3 text-sm font-semibold"
            style={{
              color: activeTab === 'question' ? ACCENT_COLOR : '#9ca3af',
              ...(activeTab === 'question' ? styles.activeTabUnderline : null),
            }}
          >
            Question Paper
          </Text>
        </Pressable>
        <Pressable onPress={() => setActiveTab('answer')}>
          <Text
            className="pb-3 text-sm font-semibold"
            style={{
              color: activeTab === 'answer' ? ACCENT_COLOR : '#9ca3af',
              ...(activeTab === 'answer' ? styles.activeTabUnderline : null),
            }}
          >
            Answer Paper
          </Text>
        </Pressable>
      </View>

      {activeTab === 'question' ? (
        <ScrollView
          className="flex-1 bg-gray-100"
          contentContainerStyle={styles.questionPaperContent}
        >
          {/* react-native-pdf's Android view has no native corner-radius
              support and fills its bounds with its own gray letterbox
              behind the page, ignoring style.borderRadius entirely — so the
              frame is rounded here and the PDF is padded well inside it,
              rather than relying on clipping the PDF's own edges. */}
          <View
            className="mx-6 mt-4 overflow-hidden rounded-3xl border border-gray-200 bg-gray-50 p-3"
            style={styles.pdfContainer}
          >
            {pdfSource && isFocused ? (
              <Pdf
                source={pdfSource}
                trustAllCerts={false}
                style={styles.pdf}
              />
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
                  <ActivityIndicator color={ACCENT_COLOR} />
                )}
              </View>
            )}
          </View>

          <View className="mx-6 mt-4 rounded-3xl bg-gray-50 p-5">
            <Text className="text-xs font-semibold tracking-wide text-gray-400">
              ASSIGNMENT DETAILS
            </Text>
            <View className="mt-1">
              <DetailRow label="Subject" value={detail.subject ?? '—'} />
              <DetailRow label="Teacher" value={detail.teacher_email} />
              <DetailRow
                label="Assigned"
                value={new Date(detail.created_at).toLocaleDateString()}
              />
              <DetailRow
                label="Due"
                value={
                  detail.due_date
                    ? new Date(detail.due_date).toLocaleDateString()
                    : 'No due date'
                }
              />
            </View>
          </View>
        </ScrollView>
      ) : (
        <>
          <View className="flex-1">
            {submissions.length > 0 ? (
              <>
                <Text className="px-6 pt-3 text-xs font-semibold tracking-wide text-gray-400">
                  SUBMISSIONS
                </Text>
                <FlatList
                  data={submissions}
                  keyExtractor={item => item.id}
                  contentContainerStyle={styles.submissionsListContent}
                  renderItem={({ item }) => (
                    <Pressable
                      onPress={() =>
                        navigation.navigate('SubmissionDetail', {
                          submissionId: item.id,
                          studentName: item.student_name,
                        })
                      }
                      className="mb-3 flex-row items-center justify-between rounded-3xl bg-gray-50 p-5"
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
              </>
            ) : (
              <View className="flex-1 items-center justify-center px-6">
                <Text className="text-sm text-gray-500">
                  No answer papers uploaded yet.
                </Text>
              </View>
            )}
          </View>

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
                <Text className="mt-2 text-xs text-gray-500">
                  Extract & upload answers
                </Text>
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
                  <Text className="mt-2 text-xs text-gray-500">
                    Scan next page
                  </Text>
                </View>
                <Pressable
                  onPress={handleFinishScanning}
                  className="mt-4 items-center"
                >
                  <Text className="text-sm font-semibold text-blue-600">
                    Done — combine into one paper
                  </Text>
                </Pressable>
              </View>
            )}
          </View>
        </>
      )}

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
  pdfContainer: {
    height: 460,
  },
  questionPaperContent: {
    paddingBottom: 32,
  },
  activeTabUnderline: {
    borderBottomWidth: 2,
    borderBottomColor: '#2f6690',
  },
  submissionsListContent: {
    paddingHorizontal: 24,
    paddingTop: 12,
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
