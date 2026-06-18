// DocType / BinderSize are defined in the domain module (the single source the
// backend parity test checks); imported for the form shapes below and
// re-exported so existing `./types` consumers keep working.
import type { DocType, BinderSize } from './domain';
export type { DocType, BinderSize };

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
