export type DocType = '1' | '2';
export type BinderSize = 1 | 3 | 5 | 7;

export interface EquipmentForm {
  eq_number: string;
  eq_doc_number: string;
  eq_doc_title: string;
  eq_doc_count: number;
  eq_doc_department: string;
  eq_doc_year: number;
}

export interface ProjectForm {
  pjt_number: string;
  pjt_test_number: string;
  pjt_doc_title: string;
  pjt_doc_writer: string;
  pjt_doc_count: number;
}

export interface LabelForm {
  docType: DocType;
  binderSize: BinderSize;
  equipment: EquipmentForm;
  project: ProjectForm;
}
