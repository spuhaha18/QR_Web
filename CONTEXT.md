# QR_Web — Binder Label Generator

사내 장비/과제 문서용 바인더 라벨 엑셀을 생성한다. 임시 워크어라운드 단계에서는 외부 프로젝트가 생성한 QR 이미지를 업로드받아 시트별로 매칭한다.

## Language

**권 (volume / doc_count)**:
하나의 논리적 문서 단위. UI 폼에서 "총 권수"로 입력하며, 권 1개당 라벨 시트 1개·QR 1개·물리 바인더 1개가 1:1:1:1로 매핑된다.
_Avoid_: 부, 문서, 카피

**시트 (sheet)**:
엑셀 워크북 안의 라벨 한 장. 권 i를 위한 시트는 B5 셀에 "i/N" 표기를 가진다.
_Avoid_: 페이지, 워크시트(코드 외 문맥에서)

**QR 이미지 (QR image)**:
외부 프로젝트가 생성한 PNG. 새 프로젝트는 `<img src="data:image/png;base64,...">` 형태로 렌더링한다. 사용자는 새 프로젝트에서 QR 이미지를 파일로 저장하여 dropzone에 드래그&드롭하거나, **우클릭 "이미지 링크 복사"** 후 data URI 입력란에 붙여넣는다. 권 i의 시트에 1개씩 삽입된다. 이 앱은 페이로드를 디코딩하지 않고 이미지로만 사용한다.
_Avoid_: QR 코드 (코드라는 표현은 페이로드 의미가 섞이므로 라벨 흐름에서는 "QR 이미지"로 통일)

**QR 이미지 입력 (QR image input)**:
라벨 앱이 QR PNG를 받는 두 가지 입력 경로.
- **파일 입력**: dropzone 드래그&드롭 또는 클릭 → 파일 선택 다이얼로그. `image/*` MIME 필터, SHA-1 중복 거부.
- **data URI 텍스트 입력**: `data:image/...;base64,...` 텍스트를 입력란에 붙여넣고 Enter 또는 '추가' 버튼 → Blob 복원 후 동일 검사 적용.
두 경로 모두 `state.images = { id, blob, hash, url }[]`에 수렴한다.
_Avoid_: 이미지 클립보드 paste(`clipboardData.items`의 `image/*`), URL fetch (서버/클라이언트 모두)
_이유_: 외부 사내 시스템 보안 정책으로 클립보드 image MIME 차단됨

**바인더 (binder)**:
권을 물리적으로 보관하는 바인더. 권당 1개.
_Avoid_: 폴더, 파일철

**바인더 사이즈 (binder size)**:
바인더의 두께. `[1, 3, 5, 7]` cm 중 하나. 시트의 QR 셀 배치/열 너비를 결정한다.
Go에서는 `label.BinderSize` 값 객체로 표현된다 — `ParseBinderSize`가 유일한 검증 경계이고(미지 사이즈/과제+1cm 거부), `ColumnWidth()`가 열 너비를 소유한다. 무효 사이즈는 어디서도 폴백되지 않는다.
_Avoid_: 두께, 폭

**doc_type**:
문서 종류. `1` = 장비(equipment), `2` = 과제(project). 폼 필드 집합과 라벨 레이아웃이 분기된다.
Go에서는 `label.DocType` 값 객체(`DocTypeEquipment`/`DocTypeProject`)로 표현된다. 분기는 `DocType` 메서드 뒤에 산다: `RequiredFields()`(필수 필드), `Layout()`(레이아웃 사실), `IsProject()`.
_Avoid_: 카테고리, 타입(단독), 문자열 "1"/"2"를 도메인 경계 밖으로 전달하기

**Layout (레이아웃 사실)**:
doc_type별 구조 사실의 단일 소스(`label.DocType.Layout()`). `QRBoxTopRow`(QR 박스 상단 행: 장비 8 / 과제 7), `HasPrintArea`(과제만 print_area), `CountCells`(i/N 마커 셀: 항상 B5, 과제는 S23 추가)를 담는다. 엑셀 제너레이터와 기하 계산이 같은 사실을 공유한다 (이전엔 매직넘버가 여러 곳에 중복).

**QRImageSet (QR 이미지 집합)**:
시트 순서로 정렬된 검증 완료 QR PNG 집합(`label.QRImageSet`). 생성 시점에 "권 1 = QR 1" 개수 불변식을 강제한다 — paste/auto 두 입력 경로 모두 이 타입을 구축하고, 제너레이터는 개수를 재검사하지 않는다. (개당 크기/PNG 유효성은 런타임 config 의존이라 HTTP intake 계층에 남는다.)

## Relationships

- **권** 1개 = **시트** 1개 = **QR 이미지** 1개 = **바인더** 1개 (`doc_count == 업로드 QR 수 == 생성 시트 수`)
- **doc_count** > 1 이면 첫 시트를 복제하여 i=2..N 시트 생성 (`_create_additional_sheets`)
- **바인더 사이즈**는 워크북 전체에 동일 적용 (시트별로 다르지 않음)
- **doc_type**은 워크북당 1개 (장비/과제 혼합 불가)

## Flagged ambiguities

- "QR 코드" vs "QR 이미지" — 라벨 흐름에서는 페이로드를 사용하지 않으므로 "QR 이미지"로 통일.
- "바인더"가 한때 "권"과 혼용 가능성 있었음 — 해소: 권은 논리 단위, 바인더는 물리 단위, 1:1.
