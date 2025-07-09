# Set Base image - Python 3.12로 업그레이드 (최신 안정 버전)
FROM python:3.12-slim

# 환경 변수 설정
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# 시스템 패키지 업데이트 및 필요한 패키지 설치
RUN apt-get update && apt-get install -y \
    --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# 비루트 사용자 생성 (보안 강화)
RUN useradd --create-home --shell /bin/bash appuser

# Set work directory
WORKDIR /app

# Copy package file
COPY requirements.txt .

# pip 업그레이드 및 패키지 설치
RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# Copy Application source
COPY . .

# uploads 디렉토리 생성 및 권한 설정
# 로그·업로드 폴더를 만들고 소유권을 실행 사용자(appuser)에게 양도
RUN mkdir -p /app/logs /app/uploads \
    && chown -R appuser:appuser /app/logs /app/uploads

# 비루트 사용자로 전환
USER appuser

# 포트 노출
EXPOSE 5000

# 헬스체크 추가
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:5000/api/health')" || exit 1

# Run Application
CMD ["python", "run_waitress.py"]
