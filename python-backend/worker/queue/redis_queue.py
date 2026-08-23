import redis

EXTRACT_INK_QUEUE = "jobs:extract_ink"
COMPOSITE_PDF_QUEUE = "jobs:composite_pdf"

EXTRACT_INK = "extract_ink"
COMPOSITE_PDF = "composite_pdf"

_QUEUE_TO_JOB_TYPE = {
    EXTRACT_INK_QUEUE: EXTRACT_INK,
    COMPOSITE_PDF_QUEUE: COMPOSITE_PDF,
}


class JobQueue:
    def __init__(self, redis_url: str, block_timeout_seconds: int = 5):
        self._client = redis.Redis.from_url(redis_url, decode_responses=True)
        self._block_timeout_seconds = block_timeout_seconds

    def next_job(self) -> tuple[str, str] | None:
        try:
            result = self._client.blpop(
                [EXTRACT_INK_QUEUE, COMPOSITE_PDF_QUEUE], timeout=self._block_timeout_seconds
            )
        except redis.exceptions.TimeoutError:
            # The socket-level read timeout can fire around the same time as
            # BLPOP's own timeout instead of Redis returning nil first; either
            # way it just means no job showed up in this poll window.
            return None
        if result is None:
            return None
        queue_name, reference_id = result
        return _QUEUE_TO_JOB_TYPE[queue_name], reference_id
