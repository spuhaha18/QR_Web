import type { EquipmentForm, ProjectForm } from './types';

export type FieldErrors = Record<string, string>;

const REQUIRED_EQ: Record<string, string> = {
  eq_number: '마스터코드를 입력하세요.',
  eq_doc_number: '문서 번호를 입력하세요.',
  eq_doc_title: '문서 제목을 입력하세요.',
  eq_doc_department: '작성 부서를 입력하세요.',
};

const REQUIRED_PJ: Record<string, string> = {
  pjt_number: '과제 코드를 입력하세요.',
  pjt_test_number: '시험 번호를 입력하세요.',
  pjt_doc_title: '문서 제목을 입력하세요.',
  pjt_doc_writer: '연구 담당자를 입력하세요.',
};

function isPositiveInt(v: unknown): boolean {
  const n = Number(v);
  return Number.isInteger(n) && n >= 1;
}

export function validateEquipment(e: EquipmentForm): FieldErrors {
  const errors: FieldErrors = {};
  for (const [k, msg] of Object.entries(REQUIRED_EQ)) {
    if (!String((e as unknown as Record<string, unknown>)[k] ?? '').trim()) errors[k] = msg;
  }
  if (!isPositiveInt(e.eq_doc_count)) errors.eq_doc_count = '권수는 1 이상이어야 합니다.';
  // 연도는 필수. 빈 값을 허용하면 서버가 조용히 현재년으로 대체해 사용자가
  // 의도하지 않은 연도의 라벨을 받게 되므로, 빈 값/비정수를 오류로 잡는다.
  if (!isPositiveInt(e.eq_doc_year)) errors.eq_doc_year = '연도를 입력하세요.';
  return errors;
}

export function validateProject(p: ProjectForm): FieldErrors {
  const errors: FieldErrors = {};
  for (const [k, msg] of Object.entries(REQUIRED_PJ)) {
    if (!String((p as unknown as Record<string, unknown>)[k] ?? '').trim()) errors[k] = msg;
  }
  if (!isPositiveInt(p.pjt_doc_count)) errors.pjt_doc_count = '권수는 1 이상이어야 합니다.';
  return errors;
}
