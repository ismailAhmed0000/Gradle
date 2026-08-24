import ReactNativeBlobUtil from 'react-native-blob-util';

export async function fetchPdfAsDataUri(url: string): Promise<string> {
  const response = await ReactNativeBlobUtil.fetch('GET', url);
  // react-native-blob-util resolves on any HTTP response, 4xx/5xx included —
  // without this check an expired presigned URL's S3 error body gets
  // base64-encoded and handed to the PDF viewer as if it were valid data,
  // which then just renders blank with no error.
  const { status } = response.info();
  if (status < 200 || status >= 300) {
    throw new Error(`Failed to fetch PDF: HTTP ${status}`);
  }
  const base64 = await response.base64();
  return `data:application/pdf;base64,${base64}`;
}
