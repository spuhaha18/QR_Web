"""
Utility functions for QR Web application
"""
import os
import time
import threading
import logging
from datetime import datetime
from flask import request

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

def delete_file_later(filepath, delay=600):
    """지정된 시간 후 파일을 삭제합니다."""
    def delete_file():
        filename = os.path.basename(filepath)
        logger.info(f"File deletion timer started for {filename} - will delete in {delay} seconds")
        time.sleep(delay)
        
        if os.path.exists(filepath):
            try:
                file_size = get_file_size_safe(filepath)
                os.remove(filepath)
                logger.info(f"File deleted successfully: {filename} (was {file_size} bytes)")
            except OSError as e:
                logger.error(f"Failed to delete file {filename}: {e}")
        else:
            logger.warning(f"File not found for deletion: {filename}")
    
    threading.Thread(target=delete_file, daemon=True).start()

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