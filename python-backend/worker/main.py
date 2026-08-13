import logging

from worker.compositing.job import run as run_composite_pdf
from worker.config import load_config
from worker.extraction.job import run as run_extract_ink
from worker.internal_client import InternalAPIError, InternalClient
from worker.queue.redis_queue import COMPOSITE_PDF, EXTRACT_INK, JobQueue
from worker.storage import S3Storage

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("worker")


def main() -> None:
    config = load_config()
    client = InternalClient(config.api_base_url, config.internal_api_token)
    storage = S3Storage(
        config.s3_endpoint_url, config.s3_access_key, config.s3_secret_key, config.s3_bucket, config.s3_region
    )
    queue = JobQueue(config.redis_url, config.blpop_timeout_seconds)

    logger.info("worker started, API base %s", config.api_base_url)
    while True:
        job = queue.next_job()
        if job is None:
            continue
        job_type, reference_id = job
        _handle_job(job_type, reference_id, client, storage)


def _handle_job(job_type: str, reference_id: str, client: InternalClient, storage: S3Storage) -> None:
    logger.info("picked up job type=%s reference_id=%s", job_type, reference_id)
    try:
        if job_type == EXTRACT_INK:
            client.start_answer_region_job(reference_id)
            run_extract_ink(reference_id, client, storage)
        elif job_type == COMPOSITE_PDF:
            client.start_composite_job(reference_id)
            run_composite_pdf(reference_id, client, storage)
        else:
            logger.error("unknown job type=%s reference_id=%s", job_type, reference_id)
            return
        logger.info("job done type=%s reference_id=%s", job_type, reference_id)
    except Exception as exc:
        logger.exception("job failed type=%s reference_id=%s", job_type, reference_id)
        _report_failure(job_type, reference_id, client, exc)


def _report_failure(job_type: str, reference_id: str, client: InternalClient, exc: Exception) -> None:
    try:
        if job_type == EXTRACT_INK:
            client.report_extraction_result(reference_id, error_message=str(exc))
        elif job_type == COMPOSITE_PDF:
            client.report_compositing_result(reference_id, error_message=str(exc))
    except InternalAPIError:
        logger.exception("failed to report failure back to API for %s %s", job_type, reference_id)


if __name__ == "__main__":
    main()
