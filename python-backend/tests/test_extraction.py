import cv2
import numpy as np

from worker.extraction.ink import extract_ink_png


def _synthetic_page() -> bytes:
    image = np.full((200, 400, 3), 245, dtype=np.uint8)
    cv2.putText(image, "hello", (40, 120), cv2.FONT_HERSHEY_SIMPLEX, 2, (20, 20, 20), 3, cv2.LINE_AA)
    ok, encoded = cv2.imencode(".jpg", image)
    assert ok
    return encoded.tobytes()


def test_extracted_png_has_alpha_channel():
    png_bytes = extract_ink_png(_synthetic_page(), crop_x=0, crop_y=0, crop_width=400, crop_height=200)
    decoded = cv2.imdecode(np.frombuffer(png_bytes, dtype=np.uint8), cv2.IMREAD_UNCHANGED)
    assert decoded.shape[2] == 4


def test_ink_pixels_are_opaque_and_background_is_transparent():
    png_bytes = extract_ink_png(_synthetic_page(), crop_x=0, crop_y=0, crop_width=400, crop_height=200)
    decoded = cv2.imdecode(np.frombuffer(png_bytes, dtype=np.uint8), cv2.IMREAD_UNCHANGED)
    alpha = decoded[:, :, 3]

    assert alpha[10:30, 10:30].max() == 0
    assert alpha[170:190, 370:390].max() == 0
    assert alpha[90:140, 40:220].max() == 255


def test_crop_is_clamped_to_image_bounds():
    png_bytes = extract_ink_png(_synthetic_page(), crop_x=-50, crop_y=-50, crop_width=10_000, crop_height=10_000)
    decoded = cv2.imdecode(np.frombuffer(png_bytes, dtype=np.uint8), cv2.IMREAD_UNCHANGED)
    assert decoded.shape[0] == 200
    assert decoded.shape[1] == 400


def _page_with_ruled_line() -> bytes:
    image = np.full((200, 400, 3), 245, dtype=np.uint8)
    cv2.putText(image, "hello", (40, 120), cv2.FONT_HERSHEY_SIMPLEX, 2, (20, 20, 20), 3, cv2.LINE_AA)
    cv2.line(image, (0, 170), (400, 170), (30, 30, 30), 1)
    ok, encoded = cv2.imencode(".jpg", image)
    assert ok
    return encoded.tobytes()


def test_ruled_line_is_removed_but_text_survives():
    png_bytes = extract_ink_png(_page_with_ruled_line(), crop_x=0, crop_y=0, crop_width=400, crop_height=200)
    decoded = cv2.imdecode(np.frombuffer(png_bytes, dtype=np.uint8), cv2.IMREAD_UNCHANGED)
    alpha = decoded[:, :, 3]

    assert alpha[165:168, 200:250].max() == 0
    assert alpha[90:140, 40:220].max() == 255


def test_long_wide_stroke_is_not_mistaken_for_a_ruled_line():
    image = np.full((200, 400, 3), 245, dtype=np.uint8)
    cv2.rectangle(image, (50, 90), (350, 110), (20, 20, 20), thickness=cv2.FILLED)
    ok, encoded = cv2.imencode(".jpg", image)
    assert ok

    png_bytes = extract_ink_png(encoded.tobytes(), crop_x=0, crop_y=0, crop_width=400, crop_height=200)
    decoded = cv2.imdecode(np.frombuffer(png_bytes, dtype=np.uint8), cv2.IMREAD_UNCHANGED)
    alpha = decoded[:, :, 3]

    assert alpha[95:105, 150:250].max() == 255


def test_isolated_specks_are_removed_but_punctuation_survives():
    image = np.full((200, 400, 3), 245, dtype=np.uint8)
    for x in range(20, 380, 20):
        for y in range(20, 80, 20):
            cv2.circle(image, (x, y), radius=1, color=(30, 30, 30), thickness=cv2.FILLED)
    cv2.circle(image, (200, 150), radius=3, color=(20, 20, 20), thickness=cv2.FILLED)
    ok, encoded = cv2.imencode(".jpg", image)
    assert ok

    png_bytes = extract_ink_png(encoded.tobytes(), crop_x=0, crop_y=0, crop_width=400, crop_height=200)
    decoded = cv2.imdecode(np.frombuffer(png_bytes, dtype=np.uint8), cv2.IMREAD_UNCHANGED)
    alpha = decoded[:, :, 3]

    assert alpha[10:90, 10:390].max() == 0
    assert alpha[145:156, 195:206].max() == 255
