import { apiClient } from './client';

export type Assignment = {
  id: string;
  owner_id: string;
  title: string;
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
  questions: Question[];
  assignment_files: AssignmentFile[];
};

export async function fetchAssignments(): Promise<Assignment[]> {
  const response = await apiClient.get<Assignment[]>('/api/assignments');
  return response.data;
}

export async function fetchAssignmentById(
  id: string,
): Promise<AssignmentDetail> {
  const response = await apiClient.get<AssignmentDetail>(
    `api/assignments/${id}`,
  );
  return response.data;
}
