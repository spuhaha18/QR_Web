# QR_Web v2.1.1.0

연구소 바인더 라벨 제작을 위한 웹 애플리케이션입니다. 직관적인 사용자 인터페이스를 통해 복잡한 과제 문서 및 기기 문서의 라벨을 표준화된 포맷으로 생성하고, Excel 파일로 손쉽게 출력할 수 있습니다.

---

## ✨ 주요 기능

- **모던 UI/UX**: Pristine Lab 테마, Deep Blue 브랜드 컬러, 다크 모드 연동 및 세련된 Toast 알림 지원
- **QR 코드 생성**: 입력된 정보(마스터코드 등)를 기반으로 자동 QR 코드 생성
- **QR 이미지 업로드**: 파일 선택 또는 드래그&드롭으로 QR 이미지 일괄 업로드
- **썸네일 미리보기**: 업로드된 QR 이미지를 썸네일로 표시하며 `<N>권` 수량 라벨 제공
- **순서 재배치**: SortableJS 기반 드래그앤드롭으로 QR 순서 자유 조정
- **표준 라벨 생성**: 연구소 바인더 규격에 맞는 폼 데이터 처리
- **Excel 출력**: 생성된 라벨 데이터를 규격화된 Excel 파일(.xlsx)로 내보내기
- **웹 인터페이스**: 반응형 디자인 및 직관적인 입력 폼 (Lucide SVG 아이콘 적용)
- **로깅 시스템**: 인앱(In-app) 로그 뷰어를 통한 실시간 시스템 상태 확인

---

## 🏗️ 프로젝트 구조

```
QR_Web/
├── app.py                    # 메인 Flask 애플리케이션 (라우트 및 폼 처리)
├── config.py                 # 설정 관리
├── utils.py                  # 유틸리티 함수들
├── document_schema.py        # 스키마·유효성 검사·라벨 데이터클래스 (EquipmentLabel, ProjectLabel)
├── label_layout.py           # 바인더 사이즈 → QR 셀 배치 설정 (get_qr_config)
├── file_lifecycle.py         # 임시 파일·디렉토리 지연 삭제 관리 (FileLifecycleManager)
├── qr_generator.py           # QR 코드 생성 모듈
├── excel_generator.py        # Excel 파일 생성 모듈
├── cache_manager.py          # 캐싱 시스템
├── performance_monitor.py    # 성능 모니터링
├── templates/                # HTML 템플릿
│   ├── index.html            # 메인 페이지 (라벨 폼)
│   ├── logs.html             # 시스템 로그 뷰어
│   └── api_docs.html         # API 명세서
├── static/                   # 정적 파일
│   ├── css/style.css         # 스타일시트
│   ├── js/qr_paste.js        # QR 업로드·드롭존·순서 정렬 UI 로직
│   └── vendor/sortablejs/    # SortableJS (드래그앤드롭 라이브러리)
├── tests/                    # 단위 테스트 (pytest)
├── uploads/                  # 생성된 파일 임시 저장소
├── logs/                     # 로그 파일
└── pyproject.toml / uv.lock  # uv 패키지 환경
```

---

## 🚀 시작하기

### 1. 사전 요구사항

- **로컬 실행**: Python 3.13 이상 및 `uv` (또는 `pip`) 패키지 관리자
- **Docker 실행**: Docker 20.10 이상 및 Docker Compose (선택사항)

### 2. 패키지 설치 방법

#### 방법 1: `uv`를 이용한 로컬 설치 (가장 권장)
```bash
git clone https://github.com/spuhaha18/QR_Web.git
cd QR_Web
uv add -r requirements.txt
```

#### 방법 2: `pip`를 이용한 로컬 설치
```bash
git clone https://github.com/spuhaha18/QR_Web.git
cd QR_Web
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt
```

#### 방법 3: Docker 사용
```bash
git clone https://github.com/spuhaha18/QR_Web.git
cd QR_Web
docker build -t qr-web:v2.1.1 .
```

#### 방법 4: Docker Compose 사용
저장소 내의 `docker-compose.yml`을 활용하여 즉시 백그라운드 구동이 가능합니다.
```bash
docker-compose up -d
```

---

## 🔧 서버 실행 방법

### 로컬 환경에서 실행
```bash
# uv 환경 사용 시
uv run app.py

# 기본 pip(가상환경) 사용 시
python app.py
```
> 서버가 정상적으로 실행되면 웹 브라우저에서 `http://localhost:5000` 으로 접속하세요.

### Docker 환경에서 실행
```bash
docker run -d --name qr-web-app -p 5000:5000 qr-web:v2.1.1
```

---

## 📚 API 명세 요약

자세한 API 스펙은 웹 애플리케이션 하단의 **API 문서** (`/api_docs.html`)에서 확인하실 수 있습니다.

### 주요 REST API 엔드포인트
- **GET** `/api/health` - 서버 상태 확인
- **POST** `/create_label` - 웹 폼 기반 라벨 생성 (QR 이미지 파일 업로드 포함)
- **POST** `/api/create_label` - API 기반 라벨 생성 및 Excel 파일 반환
- **GET** `/api/qr_image/<text>` - QR 코드 이미지(.png) 반환
- **POST** `/api/qr_image_base64` - Base64 인코딩 QR 이미지 반환
- **GET** `/api/logs` - 서버 애플리케이션 로그 조회

### POST `/api/create_label` 요청 예시 (기기 문서)
```bash
curl -X POST http://localhost:5000/api/create_label \
  -H "Content-Type: application/json" \
  -d '{
    "doc_type": "1",
    "binder_size": 5,
    "eq_number": "EQ001",
    "eq_doc_number": "DOC001",
    "eq_doc_title": "테스트 문서",
    "eq_doc_count": 3,
    "eq_doc_department": "기술부",
    "eq_doc_year": 2024
  }'
```

---

## ⚙️ 환경 변수 설정 (선택 사항)

| 변수명 | 기본값 | 설명 |
|--------|--------|------|
| `SECRET_KEY` | `'your_secret_key...'` | Flask 애플리케이션 시크릿 키 |
| `FLASK_ENV` | `'development'` | 실행 환경 (development / production) |
| `FLASK_PORT` | `5000` | 서버가 바인딩할 포트 번호 |
| `DELETE_DELAY` | `600` | 임시 생성된 파일(Excel) 삭제 타이머 (초) |

---

## 📋 버전 히스토리

### v2.1.1.0 (2026-05-19)
**아키텍처 리팩터링 및 보안 수정**
- 🏗 **모듈 분리**: 문서 스키마·유효성 검사·라벨 데이터클래스를 `document_schema.py`로 추출 (`EquipmentLabel`, `ProjectLabel`, `parse_label_request`)
- 📐 **레이아웃 설정 분리**: 바인더 사이즈 → QR 셀 배치 설정을 `label_layout.py`의 `get_qr_config()`로 이동
- 🔒 **파일 생명주기 관리**: 임시 파일 삭제 헬퍼(`delete_file_later`, `delete_dir_later`)를 관찰 가능한 `FileLifecycleManager`(`file_lifecycle.py`)로 교체
- 🛡 **경로 탐색 취약점 수정**: `/download/<filename>` 라우트에 `werkzeug.utils.safe_join` 적용
- 🚫 **QR 텍스트 길이 제한**: `/api/qr_image` 및 `/api/qr_image_base64`에 500자 상한 적용
- 🧪 **테스트 확충**: 7개 테스트 모듈, 67개 테스트 (이전 대비 15개에서 대폭 증가)

### v2.1.0 (2026-05-18)
**QR 파일 업로드·드롭존·썸네일 UX 전면 개선**
- 📂 **파일 업로드/드롭존**: QR 이미지를 파일 선택 또는 드래그&드롭으로 업로드 (기존 붙여넣기 방식 대체)
- 🖼 **썸네일 수량 라벨**: 업로드된 QR 이미지 썸네일 하단에 `<N>권` 라벨 표시
- ↕️ **드래그앤드롭 순서 재배치**: SortableJS 내장으로 QR 이미지 순서 자유 조정
- 🎨 **토스트 알림 테마화**: `showMessage` 유틸로 통합하여 일관된 스타일 적용
- 🔗 **Data URI 입력 방식 추가**: 우클릭 → 이미지 링크 복사 후 붙여넣기 지원
- 🐛 **QA 버그 수정**: syncOrder 중복 카운트(ISSUE-001), totalEl detached span(ISSUE-002) 해결

### v1.0.1.0 (2026-05-19)
**아키텍처 리팩터링 및 보안 수정**
- 🏗 **모듈 분리**: 문서 스키마·유효성 검사·라벨 데이터클래스를 `document_schema.py`로 추출 (`EquipmentLabel`, `ProjectLabel`, `parse_label_request`)
- 📐 **레이아웃 설정 분리**: 바인더 사이즈 → QR 셀 배치 설정을 `label_layout.py`의 `get_qr_config()`로 이동
- 🔒 **파일 생명주기 관리**: 임시 파일 삭제 헬퍼(`delete_file_later`, `delete_dir_later`)를 관찰 가능한 `FileLifecycleManager`(`file_lifecycle.py`)로 교체
- 🛡 **경로 탐색 취약점 수정**: `/download/<filename>` 라우트에 `werkzeug.utils.safe_join` 적용
- 🚫 **QR 텍스트 길이 제한**: `/api/qr_image` 및 `/api/qr_image_base64`에 500자 상한 적용
- 🧪 **테스트 확충**: 7개 테스트 모듈, 67개 테스트 (이전 대비 15개에서 대폭 증가)

### v1.1.0 (2026-05-11)
**UI/UX 전면 리디자인 및 시스템 안정화**
- ✨ **Pristine Lab & Deep Blue 테마**: 깔끔하고 전문적인 연구소 느낌의 UI 디자인 적용
- 🌙 **다크 모드 지원**: 시스템 설정 연동 및 수동 테마 토글 기능 추가
- 🎨 **모던 타이포그래피 및 아이콘**: Pretendard 폰트 및 Lucide SVG 아이콘 일괄 교체
- 💬 **Toast 알림**: 기존 Alert 대신 세련된 비동기 알림 UI 도입
- 🛠 **uv 환경 지원**: 최신 Python 패키지 매니저 `uv` 워크플로우 가이드 반영
- 📝 **용어 변경**: 라벨 생성 양식의 '기기 번호'를 '마스터코드'로 명칭 일괄 변경

### v1.0.0 (2025-07-10)
**초기 릴리즈**
- ✅ 기본 QR 코드 생성 및 Excel 파일 출력 기능
- ✅ 성능 모니터링, 캐싱 시스템 및 파일 정리 자동화 기능 도입

---

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 있습니다. 자세한 내용은 `LICENSE` 파일을 참조하세요.
