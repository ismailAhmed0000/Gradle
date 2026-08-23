import ReactNativeBlobUtil from 'react-native-blob-util';

export async function fetchPdfAsDataUri(url: string): Promise<string> {
  const response = await ReactNativeBlobUtil.fetch('GET', url);
  const base64 = await response.base64();
  return `data:application/pdf;base64,${base64}`;
}
