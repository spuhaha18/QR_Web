# QR_Web v1.0

연구소 바인더 라벨 제작을 위한 웹 애플리케이션

## 📋 프로젝트 개요

QR_Web은 연구소에서 사용하는 바인더 라벨을 효율적으로 생성하기 위한 웹 기반 애플리케이션입니다. QR 코드 생성, Excel 파일 출력, 그리고 직관적인 웹 인터페이스를 제공합니다.

## ✨ 주요 기능

- **QR 코드 생성**: 텍스트 입력을 통한 QR 코드 자동 생성
- **라벨 생성**: 연구소 바인더용 표준 라벨 포맷 지원
- **Excel 출력**: 생성된 라벨 데이터를 Excel 파일로 내보내기
- **웹 인터페이스**: 사용자 친화적인 웹 기반 UI
- **실시간 미리보기**: 생성 전 라벨 미리보기 기능
- **배치 처리**: 여러 라벨 동시 생성 지원

## 🏗️ 프로젝트 구조

```
QR_Web/
├── app.py                    # 메인 Flask 애플리케이션
├── app_backup.py            # 원본 앱 백업
├── config.py                # 설정 관리
├── utils.py                 # 유틸리티 함수들
├── qr_generator.py          # QR 코드 생성 모듈
├── excel_generator.py       # Excel 파일 생성 모듈
├── cache_manager.py         # 캐싱 시스템
├── performance_monitor.py   # 성능 모니터링
├── templates/               # HTML 템플릿
│   ├── index.html          # 메인 페이지
│   └── results.html        # 결과 페이지
├── static/                  # 정적 파일
│   ├── css/                # 스타일시트
│   ├── js/                 # JavaScript
│   └── images/             # 이미지 파일
├── uploads/                 # 생성된 파일 임시 저장소
├── logs/                    # 로그 파일
└── requirements.txt         # Python 의존성
```

## 🚀 설치 및 실행

### 사전 요구사항

**로컬 실행**
- Python 3.7 이상
- pip (Python 패키지 관리자)

**Docker 실행**
- Docker 20.10 이상
- Docker Compose (선택사항)

## 📦 설치 방법

### 방법 1: 로컬 설치

1. **저장소 클론**
   ```bash
   git clone https://github.com/spuhaha18/QR_Web.git
   cd QR_Web
   ```

2. **가상환경 생성 (권장)**
   ```bash
   python -m venv venv
   source venv/bin/activate  # Windows: venv\Scripts\activate
   ```

3. **의존성 설치**
   ```bash
   pip install -r requirements.txt
   ```

### 방법 2: Docker 사용

1. **저장소 클론**
   ```bash
   git clone https://github.com/spuhaha18/QR_Web.git
   cd QR_Web
   ```

2. **Docker 이미지 빌드**
   ```bash
   docker build -t qr-web:v1.0 .
   ```

3. **Docker 컨테이너 실행**
   ```bash
   docker run -d \
     --name qr-web-app \
     -p 5000:5000 \
     -e SECRET_KEY=your_secret_key_here \
     -e FLASK_ENV=production \
     qr-web:v1.0
   ```

### 방법 3: Docker Compose 사용

1. **docker-compose.yml 생성**
   ```yaml
   version: '3.8'
   
   services:
     qr-web:
       build: .
       container_name: qr-web-app
       ports:
         - "5000:5000"
       environment:
         - SECRET_KEY=your_secret_key_here
         - FLASK_ENV=production
         - LOG_LEVEL=INFO
       volumes:
         - ./logs:/app/logs
         - ./uploads:/app/uploads
       restart: unless-stopped
       healthcheck:
         test: ["CMD", "python", "-c", "import requests; requests.get('http://localhost:5000/api/health')"]
         interval: 30s
         timeout: 10s
         retries: 3
         start_period: 40s
   ```

2. **실행**
   ```bash
   docker-compose up -d
   ```

## 🔧 실행 방법

### 로컬 실행

1. **기본 실행**
   ```bash
   python app.py
   ```

2. **환경변수 설정 (선택사항)**
   ```bash
   export FLASK_ENV=development
   export SECRET_KEY=your_secret_key_here
   export LOG_LEVEL=INFO
   python app.py
   ```

3. **Waitress 서버 사용 (프로덕션 환경)**
   ```bash
   python run_waitress.py
   ```

### Docker 실행

```bash
# 백그라운드 실행
docker run -d --name qr-web-app -p 5000:5000 qr-web:v1.0

# 포그라운드 실행 (로그 확인용)
docker run --name qr-web-app -p 5000:5000 qr-web:v1.0

# 환경변수와 함께 실행
docker run -d \
  --name qr-web-app \
  -p 5000:5000 \
  -e SECRET_KEY=your_secret_key \
  -e FLASK_ENV=production \
  -v $(pwd)/logs:/app/logs \
  qr-web:v1.0
```

### 브라우저에서 접속

기본 주소: `http://localhost:5000`

## 🔧 설정

### 환경변수

| 변수명 | 기본값 | 설명 |
|--------|--------|------|
| `SECRET_KEY` | `'your_secret_key_change_in_production'` | Flask 시크릿 키 |
| `FLASK_ENV` | `'development'` | 실행 환경 |
| `FLASK_HOST` | `'0.0.0.0'` | 서버 호스트 |
| `FLASK_PORT` | `5000` | 서버 포트 |
| `LOG_LEVEL` | `'INFO'` | 로그 레벨 |
| `DELETE_DELAY` | `600` | 파일 삭제 지연 시간 (초) |
| `QR_CACHE_TTL` | `600` | QR 코드 캐시 유지 시간 (초) |

### config.py 설정

주요 설정은 `config.py` 파일에서 관리됩니다:

```python
# QR 코드 설정
QR_CODE_SIZE = 10
QR_CODE_BORDER = 4

# 파일 업로드 설정
MAX_CONTENT_LENGTH = 16 * 1024 * 1024  # 16MB

# 로그 설정
LOG_FORMAT = '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
```

## 📚 API 문서

### 웹 인터페이스

- **GET** `/` - 메인 페이지
- **POST** `/create_label` - 라벨 생성 (웹 폼)

### REST API

- **POST** `/api/create_label` - 라벨 생성 (JSON)
- **GET** `/api/qr_image/<text>` - QR 코드 이미지 생성
- **POST** `/api/qr_image_base64` - QR 코드 Base64 생성
- **GET** `/api/logs` - 로그 조회
- **GET** `/api/health` - 서버 상태 확인
- **GET** `/api/performance` - 성능 통계
- **GET** `/api/system/status` - 시스템 상태

### 사용 예시

```bash
# 라벨 생성
curl -X POST http://localhost:5000/api/create_label \
  -H "Content-Type: application/json" \
  -d '{"text": "Sample Label", "quantity": 1}'

# QR 코드 이미지 생성
curl http://localhost:5000/api/qr_image/HelloWorld

# 시스템 상태 확인
curl http://localhost:5000/api/health
```

## 🎨 사용법

### 기본 라벨 생성

1. 웹 브라우저에서 애플리케이션 접속
2. 라벨에 포함할 텍스트 입력
3. 생성할 라벨 수량 선택
4. "라벨 생성" 버튼 클릭
5. 생성된 QR 코드 및 Excel 파일 다운로드

### 배치 생성

여러 라벨을 한 번에 생성하려면:

1. CSV 파일 준비 (텍스트 데이터 포함)
2. 파일 업로드 기능 사용
3. 자동 배치 처리로 모든 라벨 생성

## 🐳 Docker 정보

### Dockerfile 특징

이 프로젝트의 Dockerfile은 다음과 같은 최적화와 보안 기능을 포함합니다:

- **Python 3.12**: 최신 안정 버전 사용
- **멀티스테이지 빌드**: 이미지 크기 최적화
- **비루트 사용자**: 보안 강화를 위한 `appuser` 사용
- **헬스체크**: 컨테이너 상태 자동 모니터링
- **환경변수 최적화**: Python 실행 환경 최적화
- **자동 디렉터리 생성**: logs, uploads 폴더 자동 설정

### Docker 명령어 참고

```bash
# 이미지 빌드
docker build -t qr-web:v1.0 .

# 컨테이너 실행
docker run -d --name qr-web-app -p 5000:5000 qr-web:v1.0

# 로그 확인
docker logs qr-web-app

# 컨테이너 상태 확인
docker ps

# 컨테이너 중지
docker stop qr-web-app

# 컨테이너 삭제
docker rm qr-web-app

# 이미지 삭제
docker rmi qr-web:v1.0
```

### 프로덕션 배포

프로덕션 환경에서는 다음과 같이 실행하는 것을 권장합니다:

```bash
docker run -d \
  --name qr-web-prod \
  -p 80:5000 \
  --restart unless-stopped \
  -e SECRET_KEY=$(openssl rand -hex 32) \
  -e FLASK_ENV=production \
  -e LOG_LEVEL=WARNING \
  -v /var/log/qr-web:/app/logs \
  -v /var/uploads/qr-web:/app/uploads \
  qr-web:v1.0
```

- **백엔드**: Python 3.7+, Flask
- **프론트엔드**: HTML5, CSS3, JavaScript
- **QR 코드 생성**: qrcode 라이브러리
- **Excel 처리**: openpyxl
- **캐싱**: Python dict 기반 메모리 캐시
- **로깅**: Python logging 모듈

## 📊 성능 특징

- **캐싱 시스템**: 중복 QR 코드 생성 방지로 속도 향상
- **메모리 관리**: 자동 가비지 컬렉션으로 메모리 최적화
- **비동기 처리**: 파일 삭제 등 백그라운드 작업
- **성능 모니터링**: 실시간 응답 시간 및 메모리 사용량 추적

## 🔒 보안 고려사항

- 환경변수를 통한 시크릿 키 관리
- 입력 데이터 검증 및 필터링
- 파일 업로드 크기 제한
- 자동 파일 정리 시스템

## 📝 로그 및 모니터링

### 로그 파일

- **위치**: `logs/` 디렉터리
- **형식**: 시간스탬프, 로그 레벨, 메시지
- **로테이션**: 일별 자동 로그 파일 생성

### 성능 모니터링

```bash
# 성능 통계 확인
curl http://localhost:5000/api/performance

# 시스템 최적화 실행
curl -X POST http://localhost:5000/api/system/optimize
```

## 🧪 테스트

### 로컬 환경 테스트

```bash
# 단위 테스트 실행
python -m pytest tests/

# 커버리지 확인
python -m pytest --cov=. tests/

# API 테스트
curl http://localhost:5000/api/health
```

### Docker 환경 테스트

```bash
# 컨테이너 헬스체크 확인
docker inspect --format='{{.State.Health.Status}}' qr-web-app

# 컨테이너 내부 접속
docker exec -it qr-web-app /bin/bash

# API 테스트
docker exec qr-web-app curl http://localhost:5000/api/health
```

## 🤝 기여 방법

1. 이 저장소를 포크합니다
2. 새로운 기능 브랜치를 생성합니다 (`git checkout -b feature/new-feature`)
3. 변경사항을 커밋합니다 (`git commit -am 'Add new feature'`)
4. 브랜치에 푸시합니다 (`git push origin feature/new-feature`)
5. Pull Request를 생성합니다

## 📋 버전 히스토리

### v1.0.0 (2025-07-10)

**초기 릴리즈**

- ✅ 기본 QR 코드 생성 기능
- ✅ 웹 인터페이스 구현
- ✅ Excel 파일 출력 기능
- ✅ REST API 제공
- ✅ 캐싱 시스템 구현
- ✅ 성능 모니터링 기능
- ✅ 로깅 시스템 구축
- ✅ 보안 강화 (입력 검증, 파일 관리)

**주요 기능**

- 단일 및 배치 라벨 생성
- 실시간 QR 코드 미리보기
- 자동 파일 정리 시스템
- 성능 최적화 (메모리 관리, 캐싱)
- 상세한 로그 및 에러 추적

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 있습니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

---
