import { describe, it, expect } from 'vitest';
import {
  DOC_TYPES,
  BINDER_SIZES,
  isProject,
  allowedBinderSizes,
  isBinderAllowed,
  requiredFields,
  REQUIRED_EQUIPMENT_FIELDS,
  REQUIRED_PROJECT_FIELDS,
  qrCountMatches,
  isReady,
} from './domain';

describe('doc type', () => {
  it('has exactly equipment and project', () => {
    expect([...DOC_TYPES]).toEqual(['1', '2']);
  });
  it('isProject only for "2"', () => {
    expect(isProject('2')).toBe(true);
    expect(isProject('1')).toBe(false);
  });
});

describe('binder size', () => {
  it('valid set is 1/3/5/7', () => {
    expect([...BINDER_SIZES]).toEqual([1, 3, 5, 7]);
  });
  it('project excludes 1cm', () => {
    expect([...allowedBinderSizes('2')]).toEqual([3, 5, 7]);
    expect(isBinderAllowed(1, '2')).toBe(false);
    expect(isBinderAllowed(3, '2')).toBe(true);
  });
  it('equipment allows all', () => {
    expect([...allowedBinderSizes('1')]).toEqual([1, 3, 5, 7]);
    expect(isBinderAllowed(1, '1')).toBe(true);
  });
});

describe('required fields', () => {
  it('equipment list', () => {
    expect([...requiredFields('1')]).toEqual([
      'eq_number',
      'eq_doc_number',
      'eq_doc_title',
      'eq_doc_count',
      'eq_doc_department',
      'eq_doc_year',
    ]);
    expect(requiredFields('1')).toBe(REQUIRED_EQUIPMENT_FIELDS);
  });
  it('project list', () => {
    expect([...requiredFields('2')]).toEqual([
      'pjt_number',
      'pjt_test_number',
      'pjt_doc_title',
      'pjt_doc_writer',
      'pjt_doc_count',
    ]);
    expect(requiredFields('2')).toBe(REQUIRED_PROJECT_FIELDS);
  });
});

describe('readiness', () => {
  it('qrCountMatches is strict equality', () => {
    expect(qrCountMatches(2, 2)).toBe(true);
    expect(qrCountMatches(3, 2)).toBe(false);
    expect(qrCountMatches(1, 2)).toBe(false);
  });
  it('isReady requires no errors, binder, matching count', () => {
    const base = { fieldErrorCount: 0, binderSelected: true, qrCount: 2, docCount: 2 };
    expect(isReady(base)).toBe(true);
    expect(isReady({ ...base, fieldErrorCount: 1 })).toBe(false);
    expect(isReady({ ...base, binderSelected: false })).toBe(false);
    expect(isReady({ ...base, qrCount: 1 })).toBe(false);
  });
});
