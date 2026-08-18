import cv2
import numpy as np

ADAPTIVE_BLOCK_SIZE = 35
ADAPTIVE_C = 15
MORPH_KERNEL_SIZE = 3

RULED_LINE_BRIDGE_GAP = 25
RULED_LINE_MIN_WIDTH_FRACTION = 0.5
RULED_LINE_MAX_HEIGHT_FRACTION = 0.05
RULED_LINE_MAX_HEIGHT_FLOOR = 6

MIN_INK_COMPONENT_AREA = 15


def extract_ink_png(image_bytes: bytes, crop_x: float, crop_y: float, crop_width: float, crop_height: float) -> bytes:
    image = cv2.imdecode(np.frombuffer(image_bytes, dtype=np.uint8), cv2.IMREAD_COLOR)
    if image is None:
        raise ValueError("could not decode source image")

    cropped = _crop(image, crop_x, crop_y, crop_width, crop_height)
    mask = _ink_mask(cropped)

    bgra = np.zeros((mask.shape[0], mask.shape[1], 4), dtype=np.uint8)
    bgra[:, :, 3] = mask

    ok, png_bytes = cv2.imencode(".png", bgra)
    if not ok:
        raise RuntimeError("failed to encode extracted ink PNG")
    return png_bytes.tobytes()


def _crop(image: np.ndarray, crop_x: float, crop_y: float, crop_width: float, crop_height: float) -> np.ndarray:
    img_h, img_w = image.shape[:2]
    x = max(0, min(int(round(crop_x)), img_w - 1))
    y = max(0, min(int(round(crop_y)), img_h - 1))
    w = max(1, min(int(round(crop_width)), img_w - x))
    h = max(1, min(int(round(crop_height)), img_h - y))
    return image[y : y + h, x : x + w]


def _ink_mask(cropped: np.ndarray) -> np.ndarray:
    gray = cv2.cvtColor(cropped, cv2.COLOR_BGR2GRAY)

    mask = cv2.adaptiveThreshold(
        gray,
        255,
        cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
        cv2.THRESH_BINARY_INV,
        blockSize=ADAPTIVE_BLOCK_SIZE,
        C=ADAPTIVE_C,
    )

    mask = cv2.bitwise_and(mask, cv2.bitwise_not(_ruled_line_mask(mask)))

    kernel = cv2.getStructuringElement(cv2.MORPH_ELLIPSE, (MORPH_KERNEL_SIZE, MORPH_KERNEL_SIZE))
    mask = cv2.morphologyEx(mask, cv2.MORPH_OPEN, kernel, iterations=1)
    mask = cv2.morphologyEx(mask, cv2.MORPH_CLOSE, kernel, iterations=2)
    mask = _remove_small_components(mask, MIN_INK_COMPONENT_AREA)
    return mask


def _remove_small_components(mask: np.ndarray, min_area: int) -> np.ndarray:
    num_labels, labels, stats, _ = cv2.connectedComponentsWithStats(mask, connectivity=8)

    cleaned = np.zeros_like(mask)
    for label in range(1, num_labels):
        if stats[label, cv2.CC_STAT_AREA] >= min_area:
            cleaned[labels == label] = 255
    return cleaned


def _ruled_line_mask(mask: np.ndarray) -> np.ndarray:
    height, width = mask.shape
    min_width = width * RULED_LINE_MIN_WIDTH_FRACTION
    max_height = max(RULED_LINE_MAX_HEIGHT_FLOOR, height * RULED_LINE_MAX_HEIGHT_FRACTION)

    bridge_kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (RULED_LINE_BRIDGE_GAP, 1))
    bridged = cv2.morphologyEx(mask, cv2.MORPH_CLOSE, bridge_kernel)

    num_labels, labels, stats, _ = cv2.connectedComponentsWithStats(bridged, connectivity=8)

    lines = np.zeros_like(mask)
    for label in range(1, num_labels):
        _, _, comp_width, comp_height, _ = stats[label]
        if comp_width >= min_width and comp_height <= max_height:
            lines[labels == label] = 255

    return cv2.dilate(lines, cv2.getStructuringElement(cv2.MORPH_ELLIPSE, (3, 3)))
