import pymupdf as fitz

PAGE_MARGIN = 50.0
LABEL_FONT_SIZE = 14.0
LABEL_HEIGHT = 30.0


def insert_ink_at_region(
    doc: "fitz.Document",
    page_number: int,
    ink_png: bytes,
    region_x: float,
    region_y: float,
    region_width: float,
    region_height: float,
) -> None:
    page = doc[page_number - 1]
    rect = fitz.Rect(region_x, region_y, region_x + region_width, region_y + region_height)
    page.insert_image(rect, stream=ink_png, keep_proportion=True)


def append_answer_page(doc: "fitz.Document", question_number: int, ink_png: bytes, page_rect: "fitz.Rect") -> int:
    page = doc.new_page(width=page_rect.width, height=page_rect.height)
    page.insert_text((PAGE_MARGIN, PAGE_MARGIN), f"Answer to Question {question_number}", fontsize=LABEL_FONT_SIZE)

    img = fitz.Pixmap(ink_png)
    available_width = page_rect.width - 2 * PAGE_MARGIN
    scale = min(1.0, available_width / img.width)
    draw_width = img.width * scale
    draw_height = img.height * scale

    top = PAGE_MARGIN + LABEL_HEIGHT
    rect = fitz.Rect(PAGE_MARGIN, top, PAGE_MARGIN + draw_width, top + draw_height)
    page.insert_image(rect, stream=ink_png)

    return doc.page_count
