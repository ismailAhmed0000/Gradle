import { apiClient } from './client';

export type AssignmentStatus = 'pending' | 'expired' | 'submitted' | 'graded';

export type Assignment = {
  id: string;
  owner_id: string;
  title: string;
  subject?: string;
  due_date?: string;
  status: AssignmentStatus;
  created_at: string;
};

export type Question = {
  id: string;
  assignment_id: string;
  assignment_file_id: string;
  question_number: number;
  has_defined_region: boolean;
  page_number?: number;
  region_x?: number;
  region_y?: number;
  region_width?: number;
  region_height?: number;
  created_at: string;
};

export type AssignmentFile = {
  id: string;
  assignment_id: string;
  file_path: string;
  page_count: number;
  created_at: string;
  download_url?: string;
};

export type AssignmentDetail = Assignment & {
  teacher_email: string;
  questions: Question[];
  assignment_files: AssignmentFile[];
};

export async function fetchAssignments(): Promise<Assignment[]> {
  const response = await apiClient.get<Assignment[]>('/api/assignments');
  return response.data;
}

export const ASSIGNMENT_STATUS_LABELS: Record<AssignmentStatus, string> = {
  pending: 'Pending',
  expired: 'Expired',
  submitted: 'Submitted',
  graded: 'Graded',
};

export const ASSIGNMENT_STATUS_COLORS: Record<AssignmentStatus, string> = {
  pending: '#9ca3af',
  expired: '#dc2626',
  submitted: '#d97706',
  graded: '#16a34a',
};

export async function fetchAssignmentById(
  id: string,
): Promise<AssignmentDetail> {
  const response = await apiClient.get<AssignmentDetail>(
    `api/assignments/${id}`,
  );
  return response.data;
}
