import cv2
import numpy as np
import pymupdf as fitz

from worker.compositing.pdf import append_answer_page, insert_ink_at_region


def _ink_png(width=300, height=100) -> bytes:
    rgba = np.zeros((height, width, 4), dtype=np.uint8)
    rgba[20:40, 20:200, 3] = 255  # a solid ink stroke, fully opaque
    ok, encoded = cv2.imencode(".png", rgba)
    assert ok
    return encoded.tobytes()


def _blank_doc() -> "fitz.Document":
    doc = fitz.open()
    doc.new_page(width=612, height=792)
    return doc


def _render_pixels(page: "fitz.Page") -> np.ndarray:
    pix = page.get_pixmap()
    return np.frombuffer(pix.samples, dtype=np.uint8).reshape(pix.height, pix.width, pix.n)


def test_insert_ink_at_region_changes_pixels_only_inside_rect():
    doc = _blank_doc()
    before = _render_pixels(doc[0]).copy()

    insert_ink_at_region(doc, page_number=1, ink_png=_ink_png(), region_x=100, region_y=100, region_width=300, region_height=100)

    after = _render_pixels(doc[0])
    assert before.shape == after.shape
    assert not np.array_equal(before, after)

    outside_before = before[500:520, 500:520]
    outside_after = after[500:520, 500:520]
    assert np.array_equal(outside_before, outside_after)
    doc.close()


def test_append_answer_page_adds_a_page_sized_like_the_original():
    doc = _blank_doc()
    original_rect = doc[0].rect
    assert doc.page_count == 1

    new_page_number = append_answer_page(doc, question_number=3, ink_png=_ink_png(), page_rect=original_rect)

    assert new_page_number == 2
    assert doc.page_count == 2
    new_page = doc[new_page_number - 1]
    assert new_page.rect.width == original_rect.width
    assert new_page.rect.height == original_rect.height
    assert "Answer to Question 3" in new_page.get_text()
    doc.close()


def test_append_answer_page_scales_wide_ink_down_to_fit():
    doc = _blank_doc()
    original_rect = doc[0].rect
    wide_ink = _ink_png(width=4000, height=200)

    append_answer_page(doc, question_number=1, ink_png=wide_ink, page_rect=original_rect)

    images = doc[1].get_image_info()
    assert len(images) == 1
    placed_width = images[0]["bbox"][2] - images[0]["bbox"][0]
    assert placed_width <= original_rect.width - 2 * 50.0 + 1e-6
    doc.close()
