import { useEffect, useMemo, useRef, useState } from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import Pdf from 'react-native-pdf';
import { TasksStackParamList } from '../navigation/types';
import { useSubmissions } from '../context/SubmissionsContext';
import { fetchPdfAsDataUri } from '../utils/pdf';

type Props = NativeStackScreenProps<TasksStackParamList, 'SubmissionDetail'>;

export function SubmissionDetailScreen({ route }: Props) {
  const { submissionId, studentName } = route.params;
  const {
    compositedBySubmissionId,
    startPollingComposited,
    stopPollingComposited,
  } = useSubmissions();
  const result = compositedBySubmissionId[submissionId];

  const [pdfDataUri, setPdfDataUri] = useState<string | null>(null);
  const [downloadFailed, setDownloadFailed] = useState(false);
  // download_url is a presigned S3 link that expires after ~15 minutes;
  // polling stops once the document is done, so a screen left open past
  // that window would hold a dead URL. Retry once by polling again (which
  // mints a fresh URL) before surfacing an error.
  const pdfRetriedRef = useRef(false);

  useEffect(() => {
    startPollingComposited(submissionId);
    return () => stopPollingComposited(submissionId);
  }, [submissionId, startPollingComposited, stopPollingComposited]);

  useEffect(() => {
    if (result?.status !== 'done' || !result.download_url) {
      return;
    }
    let cancelled = false;
    setDownloadFailed(false);
    fetchPdfAsDataUri(result.download_url)
      .then(dataUri => {
        if (!cancelled) setPdfDataUri(dataUri);
      })
      .catch(() => {
        if (cancelled) {
          return;
        }
        if (!pdfRetriedRef.current) {
          pdfRetriedRef.current = true;
          startPollingComposited(submissionId);
        } else {
          setDownloadFailed(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [result?.status, result?.download_url, submissionId, startPollingComposited]);

  const pdfSource = useMemo(
    () => (pdfDataUri ? { uri: pdfDataUri } : null),
    [pdfDataUri],
  );

  if (!result) {
    return (
      <View className="flex-1 items-center justify-center bg-white">
        <ActivityIndicator size="large" color="#2f6690" />
      </View>
    );
  }

  return (
    <View className="flex-1 bg-white">
      <View className="px-6 pt-6">
        <Text className="text-2xl font-bold text-gray-900">{studentName}</Text>
      </View>

      <View className="mt-4 flex-1 bg-gray-100">
        {pdfSource ? (
          <Pdf source={pdfSource} trustAllCerts={false} style={styles.pdf} />
        ) : (
          <View className="flex-1 items-center justify-center px-6">
            {result.status === 'failed' ? (
              <Text className="text-center text-sm text-red-600">
                {result.error_message ?? 'Grading failed for this submission.'}
              </Text>
            ) : downloadFailed ? (
              <Text className="text-center text-sm text-red-600">
                Failed to download the composited document.
              </Text>
            ) : (
              <>
                <ActivityIndicator color="#2f6690" />
                <Text className="mt-3 text-center text-sm text-gray-500">
                  Composited document isn't ready yet — this updates
                  automatically once grading finishes.
                </Text>
              </>
            )}
          </View>
        )}
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
