"""
Utility functions for QR Web application
"""
import os
import io
import time
import threading
import logging
from datetime import datetime
from flask import request
from PIL import Image as PILImage

logger = logging.getLogger(__name__)

def get_client_info():
    """클라이언트 IP와 User-Agent 정보를 반환합니다."""
    client_ip = request.environ.get('HTTP_X_FORWARDED_FOR', 
                                   request.environ.get('REMOTE_ADDR', 'unknown'))
    user_agent = request.environ.get('HTTP_USER_AGENT', 'unknown')
    return client_ip, user_agent

def log_client_access(page_name, additional_info=None):
    """클라이언트 접근을 로그에 기록합니다."""
    client_ip, user_agent = get_client_info()
    user_agent_truncated = user_agent[:100] + ('...' if len(user_agent) > 100 else '')
    
    log_msg = f"{page_name} accessed from {client_ip} - User-Agent: {user_agent_truncated}"
    if additional_info:
        log_msg += f" - {additional_info}"
    
    logger.info(log_msg)

def create_directory_if_not_exists(directory):
    """디렉토리가 존재하지 않으면 생성합니다."""
    if not os.path.exists(directory):
        os.makedirs(directory)
        logger.info(f"Created directory: {directory}")

def get_file_size_safe(filepath):
    """파일 크기를 안전하게 가져옵니다."""
    try:
        return os.path.getsize(filepath) if os.path.exists(filepath) else 0
    except OSError as e:
        logger.warning(f"Could not get file size for {filepath}: {e}")
        return 0

def generate_timestamp_filename(base_name, extension="xlsx"):
    """타임스탬프가 포함된 파일명을 생성합니다."""
    timestamp = datetime.now().strftime('%Y%m%d%H%M%S')
    return f"{base_name}_{timestamp}.{extension}"

def validate_and_clean_input(value, default=""):
    """입력값을 검증하고 정리합니다."""
    if not value:
        return default
    return str(value).strip().replace("\n", "").replace("\r", "")

def safe_int_conversion(value, default=1):
    """문자열을 안전하게 정수로 변환합니다."""
    if not value:
        return default
    try:
        result = int(value) if str(value).isdigit() else default
        return max(1, result)  # 최소값 1 보장
    except (ValueError, TypeError):
        return default


def validate_qr_image_bytes(data: bytes) -> bool:
    """PNG 바이트 유효성 검사. Pillow verify + PNG 형식 강제."""
    if not data:
        return False
    try:
        img = PILImage.open(io.BytesIO(data))
        img.verify()          # 파일 손상/위조 검사 (verify 후 img 재사용 불가)
        img2 = PILImage.open(io.BytesIO(data))
        return img2.format == 'PNG'
    except Exception:
        return False


