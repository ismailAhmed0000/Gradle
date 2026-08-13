from worker.extraction.ink import extract_ink_png
from worker.internal_client import InternalClient
from worker.storage import S3Storage

INK_CONTENT_TYPE = "image/png"


def run(answer_region_id: str, client: InternalClient, storage: S3Storage) -> None:
    context = client.get_answer_region_context(answer_region_id)
    raw_image = storage.download_bytes(context["source_page"]["raw_image_path"])

    ink_png = extract_ink_png(
        raw_image,
        context["crop_x"],
        context["crop_y"],
        context["crop_width"],
        context["crop_height"],
    )

    key = f"answer-regions/{answer_region_id}/ink.png"
    storage.upload_bytes(key, ink_png, INK_CONTENT_TYPE)
    client.report_extraction_result(answer_region_id, extracted_ink_path=key)
