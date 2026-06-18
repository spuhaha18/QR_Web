package label

import "testing"

// Ported from tests/test_label_layout.py (TestGetQrConfig), updated for the
// BinderSize value object. Column width now lives on the validated type;
// invalid sizes are rejected at ParseBinderSize, not silently defaulted.

func TestBinderSize_ColumnWidths(t *testing.T) {
	cases := map[BinderSize]float64{7: 1.875, 5: 1.25, 3: 1.0, 1: 0.75}
	for size, want := range cases {
		if got := size.ColumnWidth(); got != want {
			t.Errorf("BinderSize(%d).ColumnWidth() = %v, want %v", int(size), got, want)
		}
	}
}

func TestParseBinderSize_AcceptsValidEquipmentSizes(t *testing.T) {
	for _, raw := range []string{"1", "3", "5", "7"} {
		b, err := ParseBinderSize(raw, DocTypeEquipment)
		if err != nil {
			t.Errorf("ParseBinderSize(%q, equipment): unexpected error %v", raw, err)
		}
		if b.Int() == 0 {
			t.Errorf("ParseBinderSize(%q): zero value", raw)
		}
	}
}

func TestParseBinderSize_RejectsUnknownSize(t *testing.T) {
	_, err := ParseBinderSize("99", DocTypeEquipment)
	if err == nil {
		t.Fatal("expected error for unknown binder size 99, got nil")
	}
	if ValidationMessage(err) != "잘못된 바인더 크기입니다." {
		t.Errorf("message = %q, want 잘못된 바인더 크기입니다.", ValidationMessage(err))
	}
}

func TestParseBinderSize_RejectsNonNumeric(t *testing.T) {
	if _, err := ParseBinderSize("bad", DocTypeEquipment); err == nil {
		t.Fatal("expected error for non-numeric binder size, got nil")
	}
}

func TestParseBinderSize_ProjectRejects1cm(t *testing.T) {
	_, err := ParseBinderSize("1", DocTypeProject)
	if err == nil {
		t.Fatal("expected error rejecting 1cm for project, got nil")
	}
	want := "과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다."
	if ValidationMessage(err) != want {
		t.Errorf("message = %q, want %q", ValidationMessage(err), want)
	}
}
