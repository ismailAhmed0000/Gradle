import requests


class InternalAPIError(Exception):
    pass


class InternalClient:
    def __init__(self, base_url: str, token: str, timeout: float = 30.0):
        self._base_url = base_url.rstrip("/")
        self._headers = {"X-Internal-Token": token}
        self._timeout = timeout

    def _request(self, method: str, path: str, **kwargs) -> dict | None:
        resp = requests.request(
            method, f"{self._base_url}{path}", headers=self._headers, timeout=self._timeout, **kwargs
        )
        if resp.status_code >= 400:
            raise InternalAPIError(f"{method} {path} -> {resp.status_code}: {resp.text}")
        return resp.json() if resp.content else None

    def get_answer_region_context(self, answer_region_id: str) -> dict:
        return self._request("GET", f"/internal/answer-regions/{answer_region_id}/context")

    def start_answer_region_job(self, answer_region_id: str) -> None:
        self._request("PATCH", f"/internal/answer-regions/{answer_region_id}/start")

    def report_extraction_result(
        self, answer_region_id: str, *, extracted_ink_path: str | None = None, error_message: str | None = None
    ) -> None:
        if error_message is not None:
            body = {"status": "failed", "error_message": error_message}
        else:
            body = {"status": "done", "extracted_ink_path": extracted_ink_path}
        self._request("PATCH", f"/internal/answer-regions/{answer_region_id}", json=body)

    def get_composite_context(self, composited_document_id: str) -> dict:
        return self._request("GET", f"/internal/composited-documents/{composited_document_id}/context")

    def start_composite_job(self, composited_document_id: str) -> None:
        self._request("PATCH", f"/internal/composited-documents/{composited_document_id}/start")

    def report_compositing_result(
        self,
        composited_document_id: str,
        *,
        file_path: str | None = None,
        pages: list | None = None,
        error_message: str | None = None,
    ) -> None:
        if error_message is not None:
            body = {"status": "failed", "error_message": error_message}
        else:
            body = {"status": "done", "file_path": file_path, "pages": pages}
        self._request("PATCH", f"/internal/composited-documents/{composited_document_id}", json=body)
