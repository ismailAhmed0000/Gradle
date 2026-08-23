import { apiClient } from './client';

export type Submission = {
  id: string;
  assignment_id: string;
  student_name: string;
  status: 'pending' | 'processing' | 'composited' | 'failed';
  created_at: string;
};

export type SubmissionPage = {
  id: string;
  submission_id: string;
  page_number: number;
  raw_image_path: string;
  created_at: string;
};

export type UploadPageResult = {
  page: SubmissionPage;
  answer_regions_queued: number;
};

export type SubmissionSummary = Submission & {
  page_count: number;
  answer_regions_done: number;
  answer_regions_total: number;
};

export type CompositedResult = {
  status: 'pending' | 'generating' | 'done' | 'failed';
  download_url?: string;
  error_message?: string;
};

export async function listSubmissions(
  assignmentId: string,
): Promise<SubmissionSummary[]> {
  const response = await apiClient.get<SubmissionSummary[]>(
    `/api/assignments/${assignmentId}/submissions`,
  );
  return response.data;
}

export async function fetchCompositedDocument(
  submissionId: string,
): Promise<CompositedResult> {
  const response = await apiClient.get<CompositedResult>(
    `/api/submissions/${submissionId}/composited`,
  );
  return response.data;
}

export async function createSubmission(
  assignmentId: string,
  studentName: string,
): Promise<Submission> {
  const response = await apiClient.post<Submission>(
    `/api/assignments/${assignmentId}/submissions`,
    { student_name: studentName },
  );
  return response.data;
}

export async function uploadSubmissionPage(
  submissionId: string,
  pageNumber: number,
  photo: { uri: string; fileName: string; type: string },
): Promise<UploadPageResult> {
  const form = new FormData();
  form.append('page_number', String(pageNumber));
  form.append('page', {
    uri: photo.uri,
    name: photo.fileName,
    type: photo.type,
  } as unknown as Blob);

  const response = await apiClient.post<UploadPageResult>(
    `/api/submissions/${submissionId}/pages`,
    form,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  );
  return response.data;
}
