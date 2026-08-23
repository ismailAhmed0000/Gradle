import { launchCamera } from 'react-native-image-picker';

export type CapturedPage = {
  uri: string;
  fileName: string;
  type: string;
};

const MAX_DIMENSION = 2000;
const JPEG_QUALITY = 0.7;

export async function capturePage(): Promise<CapturedPage | null> {
  const result = await launchCamera({
    mediaType: 'photo',
    cameraType: 'back',
    quality: JPEG_QUALITY,
    maxWidth: MAX_DIMENSION,
    maxHeight: MAX_DIMENSION,
    saveToPhotos: false,
    includeBase64: false,
  });

  if (result.didCancel || result.errorCode || !result.assets?.length) {
    return null;
  }

  const asset = result.assets[0];
  if (!asset.uri) {
    return null;
  }

  return {
    uri: asset.uri,
    fileName: asset.fileName ?? `page-${Date.now()}.jpg`,
    type: asset.type ?? 'image/jpeg',
  };
}
