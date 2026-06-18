// Single source of the label domain rules on the frontend.
//
// The Go backend (internal/label) owns the canonical rules; these constants and
// predicates mirror them so the UI validates the same way the server does. A Go
// parity test (internal/label/contract_parity_test.go) parses this file and
// fails if the doc types, binder sizes, or required-field lists drift from the
// backend value objects — that test is the single enforcement that keeps the
// two languages in sync.

export type DocType = '1' | '2';

export const DOC_TYPE_EQUIPMENT: DocType = '1';
export const DOC_TYPE_PROJECT: DocType = '2';
export const DOC_TYPES: readonly DocType[] = ['1', '2'];

export function isProject(d: DocType): boolean {
  return d === DOC_TYPE_PROJECT;
}

export type BinderSize = 1 | 3 | 5 | 7;

export const BINDER_SIZES: readonly BinderSize[] = [1, 3, 5, 7];

// 과제(project) 문서는 1cm 바인더를 사용할 수 없다 — 백엔드 ParseBinderSize와 동일.
export function allowedBinderSizes(d: DocType): readonly BinderSize[] {
  return isProject(d) ? BINDER_SIZES.filter((b) => b !== 1) : BINDER_SIZES;
}

export function isBinderAllowed(b: BinderSize, d: DocType): boolean {
  return allowedBinderSizes(d).includes(b);
}

// 필수 필드 — 백엔드 label.EquipmentRequiredFields / ProjectRequiredFields와
// 동일한 집합·순서. (per-field 한국어 메시지는 UX 관심사로 validation.ts가 소유.)
export const REQUIRED_EQUIPMENT_FIELDS: readonly string[] = [
  'eq_number',
  'eq_doc_number',
  'eq_doc_title',
  'eq_doc_count',
  'eq_doc_department',
  'eq_doc_year',
];

export const REQUIRED_PROJECT_FIELDS: readonly string[] = [
  'pjt_number',
  'pjt_test_number',
  'pjt_doc_title',
  'pjt_doc_writer',
  'pjt_doc_count',
];

export function requiredFields(d: DocType): readonly string[] {
  return isProject(d) ? REQUIRED_PROJECT_FIELDS : REQUIRED_EQUIPMENT_FIELDS;
}

// 권 = QR 1:1 — 업로드(또는 생성된) QR 수가 권수와 정확히 일치해야 한다.
// 백엔드 BuildQRImageSet의 개수 불변식과 같은 규칙의 클라이언트 측 표현.
export function qrCountMatches(qrCount: number, docCount: number): boolean {
  return qrCount === docCount;
}

// 제출 가능 판정 — 폼 오류 0 + 바인더 선택됨 + 권=QR. 프레젠테이션(App.svelte)
// 밖의 순수 함수로, HTTP/DOM 없이 단위 테스트 가능.
export function isReady(opts: {
  fieldErrorCount: number;
  binderSelected: boolean;
  qrCount: number;
  docCount: number;
}): boolean {
  return (
    opts.fieldErrorCount === 0 &&
    opts.binderSelected &&
    qrCountMatches(opts.qrCount, opts.docCount)
  );
}
