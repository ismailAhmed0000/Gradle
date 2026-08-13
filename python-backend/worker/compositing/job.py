import pymupdf as fitz

from worker.compositing.pdf import append_answer_page, insert_ink_at_region
from worker.internal_client import InternalClient
from worker.storage import S3Storage

PDF_CONTENT_TYPE = "application/pdf"


def run(composited_document_id: str, client: InternalClient, storage: S3Storage) -> None:
    context = client.get_composite_context(composited_document_id)
    submission_id = context["submission_id"]
    version = context["version"]
    answers = context["answers"]

    assignment_files = {f["id"]: f for f in context["assignment_files"]}
    primary_file_id = answers[0]["assignment_file_id"] if answers else next(iter(assignment_files))
    pdf_bytes = storage.download_bytes(assignment_files[primary_file_id]["file_path"])

    doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    pages_report = [
        {"page_number": i + 1, "page_type": "original_with_overlay", "question_id": None}
        for i in range(doc.page_count)
    ]
    first_page_rect = doc[0].rect

    for answer in answers:
        ink_png = storage.download_bytes(answer["extracted_ink_path"])
        if answer["has_defined_region"]:
            insert_ink_at_region(
                doc,
                answer["page_number"],
                ink_png,
                answer["region_x"],
                answer["region_y"],
                answer["region_width"],
                answer["region_height"],
            )
            pages_report[answer["page_number"] - 1]["question_id"] = answer["question_id"]
        else:
            new_page_number = append_answer_page(doc, answer["question_number"], ink_png, first_page_rect)
            pages_report.append(
                {"page_number": new_page_number, "page_type": "appended_blank", "question_id": answer["question_id"]}
            )

    output_bytes = doc.tobytes()
    doc.close()

    key = f"composited/{submission_id}/v{version}.pdf"
    storage.upload_bytes(key, output_bytes, PDF_CONTENT_TYPE)
    client.report_compositing_result(composited_document_id, file_path=key, pages=pages_report)
