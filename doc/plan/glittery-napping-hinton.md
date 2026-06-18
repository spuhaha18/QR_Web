# 사용 설명서 팝업 기능

## Context

메인 화면(`templates/index.html`)에 신규 사용자가 참고할 **사용 설명서**가 없다. 현재 설명서는
`DSMS 문서 등록 및 라벨 출력 매뉴얼.pptx` 파일로만 존재 — 텍스트 없는 이미지 슬라이드 5장
(표지 + 단계별 주석 스크린샷 + Activity 표)이라 웹에서 바로 못 본다.

목표: 메인 헤더에 "사용 설명서" 링크 추가 → 클릭 시 **팝업(모달)** 으로 설명서 내용을 표시.
PPT를 그대로 박는 대신 사이트 테마(Deep Blue / 다크모드)에 맞춘 **네이티브 HTML로 재구성**해
예쁘고 반응형·검색가능하게 만든다.

## 설계 결정 (확정)

- **구현 방식**: 네이티브 HTML 재구성. 스크린샷은 이미지, 제목·단계·표·번호는 HTML/CSS.
- **레이아웃**: 좌측 목차(TOC) + 우측 본문. 목차 클릭 시 해당 섹션 스크롤/표시.
- **단계 번호**: PPT의 빨간 동그라미 폐기. 깨끗한 스크린샷 위에 HTML/CSS 배지(①②③)를
  절대위치(%) 오버레이 + Activity 표 각 행에 매칭 배지.
- **스크린샷 출처**:
  - DSMS 외부 화면(슬라이드 2 등록, 4 QR보기 팝업) → **사용자가 번호 없는 깨끗한 PNG 제공** (구현 전 선행 필요)
  - 우리 라벨 앱 화면(슬라이드 3 입력, 5 라벨만들기) → **Playwright로 `https://label.inno-n.duckdns.org` 재캡처** (라이트/다크 각각)

## 콘텐츠 구조 (논리 흐름으로 재정렬)

| 섹션 | 출처 슬라이드 | 내용 |
|------|------|------|
| 개요/시작하기 | 표지(1) | 매뉴얼 소개, 전체 4단계 흐름 안내 |
| 1. DSMS 문서 등록 | 2 | 8단계 + Activity 표 (DSMS 스크린샷) |
| 2. QR 코드 복사 | 4 | DSMS에서 QR보기 → 이미지 링크 복사 (2단계) |
| 3. 라벨 정보 입력 | 3 | 라벨앱 접속 → 기본 설정 → 기기/과제 문서 정보 입력 |
| 4. 라벨 생성 | 5 | 이미지 URI 붙여넣기 → 추가 → 라벨 만들기 + 바인더 2개 이상 케이스 |

## 변경 파일

**`app.py`** — `/api/docs` 라우트(line ~358-362) 다음에 `/manual` 추가. 기존 페이지 라우트와
동일 패턴:
```python
@app.route('/manual')
def manual_page():
    return render_template('manual.html')
```
설명서 본문을 fetch로 모달에 주입하므로 `manual.html`은 `<html>` 래퍼 없는 프래그먼트
(루트 `<div class="manual">` … TOC + 섹션들)로 작성.

**`templates/manual.html`** (신규) — 설명서 본문 프래그먼트. TOC(`<nav class="manual-toc">`)
+ 섹션들(`<section id="step-1">` …). 각 단계는 `figure` 안에 스크린샷 `<img>` + 절대위치
배지(`<span class="step-badge" style="top:..%;left:..%">①</span>`), 그 아래 Activity `<table>`.
이미지 경로는 `{{ url_for('static', filename='img/manual/...') }}`. 다크모드는 `<img>`에
`data-light`/`data-dark` 두 소스 두고 JS로 토글(테마 연동).

**`templates/index.html`**:
- 헤더 링크 추가 — `.header-links`(line 199-208) 안, `/api/docs` 링크 옆에
  `<button class="header-link" onclick="openManual()"><i data-lucide="book-open"></i>사용 설명서</button>`
  (기존 `.header-link` 스타일 그대로 재사용).
- 모달 마크업 추가 — `<div class="manual-modal" id="manualModal">`(backdrop + 카드 + 닫기버튼 +
  본문 컨테이너 `#manualBody`).
- `<script>` 블록(기존 inline JS 패턴)에 `openManual()`/`closeManual()` 추가:
  최초 1회 `fetch('/manual')` → `#manualBody.innerHTML` 주입 후 캐시, `lucide.createIcons()`
  재호출, 모달 표시. ESC/백드롭 클릭/✕ 닫기. 목차 클릭 → 해당 섹션 스크롤.

**`static/css/style.css`** — 기존 테마 변수(`--primary-color` `#1e3a8a`/dark `#3b82f6`,
`--surface-color`, `--text-main`, `--radius-*`, `--shadow-*`) 재사용해 추가:
- `.manual-modal`(고정 오버레이, z-index 9999 — 기존 `.toast-container`와 충돌 없게), 백드롭 블러
- `.manual-card`(중앙 카드, max-width ~960px, 높이 85vh, 둥근 모서리, 그림자), 열림 애니메이션
- `.manual-toc`(좌측, sticky), `.manual-content`(우측, 스크롤), 2열 그리드 → 모바일 1열 스택
- `.manual-section`, `.manual-figure`(position:relative), `.step-badge`(원형 배지, primary 배경)
- `.manual-table`(No./Activity 표, 줄무늬, 강조행)
- 다크모드 자동 적용(변수 기반). 반응형(`@media max-width:768px`).

**`static/img/manual/`** (신규 디렉토리) — 스크린샷 저장:
- `dsms-register.png` (사용자 제공, DSMS 등록)
- `dsms-qr.png` (사용자 제공, QR보기 팝업)
- `app-input-light.png` / `app-input-dark.png` (Playwright 캡처, 라벨 입력 화면)
- `app-create-light.png` / `app-create-dark.png` (Playwright 캡처, 라벨 만들기 화면)

## 재사용 (신규 코드 최소화)

- 헤더 링크: 기존 `.header-link` 클래스 그대로 (style.css:167-193).
- 모달 표시 패턴: 기존 Toast(`showMessage`, index.html:126-161)의 fixed+애니메이션 관례 차용.
- 테마: 기존 CSS 변수 + `data-theme` 토글 로직(index.html `toggleTheme`) 재사용. 모달 이미지
  다크/라이트 전환도 같은 `data-theme` 감지.
- 라우트: `/logs`·`/api/docs` 동일 `render_template` 패턴.

## 선행 조건

구현 전 사용자로부터 **깨끗한 DSMS 스크린샷 2장**(등록 화면, QR보기 팝업, 빨간 번호 없는 버전)
수령 필요. 수령 전까지 임시 플레이스홀더로 레이아웃 구성 가능.

## 검증

1. `python main.py`(또는 `run_waitress.py`)로 앱 실행.
2. Playwright로 `/` 접속 → 헤더 "사용 설명서" 클릭 → 모달 열림 확인.
3. TOC 4개 섹션 클릭 → 해당 섹션 이동 확인.
4. 스크린샷 로드 + 배지 위치 정확 확인(스크린샷). Activity 표 번호 매칭 확인.
5. 다크모드 토글 → 모달·이미지 다크 버전 전환 확인.
6. ESC / 백드롭 / ✕ 닫기 동작.
7. 모바일 폭(@375px) → 목차 위, 본문 아래 1열 스택 반응형 확인.
8. Playwright 스크린샷으로 라이트/다크 최종 비주얼 점검.
