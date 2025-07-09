"""
Configuration management for QR Web application
"""
import os
from datetime import timedelta

class Config:
    """기본 설정 클래스"""
    
    # Flask 설정
    SECRET_KEY = os.environ.get('SECRET_KEY', 'your_secret_key_change_in_production')
    DEBUG = os.environ.get('FLASK_DEBUG', 'False').lower() == 'true'
    HOST = os.environ.get('FLASK_HOST', '0.0.0.0')
    PORT = int(os.environ.get('FLASK_PORT', 5000))
    
    # 디렉토리 설정
    UPLOAD_FOLDER = os.environ.get('UPLOAD_FOLDER', 'uploads')
    LOG_FOLDER = os.environ.get('LOG_FOLDER', 'logs')
    
    # 파일 관리 설정
    DELETE_DELAY = int(os.environ.get('DELETE_DELAY', 600))  # 초 단위
    MAX_CONTENT_LENGTH = int(os.environ.get('MAX_CONTENT_LENGTH', 16 * 1024 * 1024))  # 16MB
    
    # 로깅 설정
    LOG_LEVEL = os.environ.get('LOG_LEVEL', 'INFO')
    LOG_FORMAT = '%(asctime)s %(levelname)s %(name)s %(threadName)s : %(message)s'
    
    # QR 코드 설정
    QR_BOX_SIZE = int(os.environ.get('QR_BOX_SIZE', 10))
    QR_BORDER = int(os.environ.get('QR_BORDER', 2))
    QR_CACHE_TTL = int(os.environ.get('QR_CACHE_TTL', 600))  # 초 단위
    
    # 성능 설정
    PERFORMANCE_MONITORING = os.environ.get('PERFORMANCE_MONITORING', 'True').lower() == 'true'
    CACHE_ENABLED = os.environ.get('CACHE_ENABLED', 'True').lower() == 'true'
    MEMORY_OPTIMIZATION_INTERVAL = int(os.environ.get('MEMORY_OPTIMIZATION_INTERVAL', 3600))  # 초 단위
    
    # API 설정
    API_RATE_LIMIT = os.environ.get('API_RATE_LIMIT', '100/hour')
    
    # 보안 설정
    ALLOWED_EXTENSIONS = {'xlsx', 'png', 'jpg', 'jpeg'}
    MAX_FILENAME_LENGTH = 100
    
    # 문서 타입 설정
    DOCUMENT_TYPES = {
        '1': 'equipment',
        '2': 'project'
    }
    
    BINDER_SIZES = [1, 3, 5, 7]  # cm 단위
    
    # Excel 설정
    EXCEL_FONT_NAME = 'times new roman'
    EXCEL_DEFAULT_FONT_SIZE = 12
    EXCEL_TITLE_FONT_SIZE = 16
    
    @classmethod
    def validate_config(cls):
        """설정 유효성을 검증합니다."""
        errors = []
        
        # 필수 디렉토리 확인
        for folder in [cls.UPLOAD_FOLDER, cls.LOG_FOLDER]:
            if not os.path.exists(folder):
                try:
                    os.makedirs(folder)
                except OSError as e:
                    errors.append(f"Cannot create directory {folder}: {e}")
        
        # 숫자 설정 유효성 검증
        if cls.DELETE_DELAY < 60:
            errors.append("DELETE_DELAY should be at least 60 seconds")
        
        if cls.QR_CACHE_TTL < 60:
            errors.append("QR_CACHE_TTL should be at least 60 seconds")
        
        if errors:
            raise ValueError(f"Configuration errors: {', '.join(errors)}")
        
        return True

class DevelopmentConfig(Config):
    """개발 환경 설정"""
    DEBUG = True
    LOG_LEVEL = 'DEBUG'
    PERFORMANCE_MONITORING = True

class ProductionConfig(Config):
    """운영 환경 설정"""
    DEBUG = False
    LOG_LEVEL = 'INFO'
    DELETE_DELAY = 300  # 5분
    SECRET_KEY = os.environ.get('SECRET_KEY')  # 반드시 환경변수에서 설정
    
    @classmethod
    def validate_config(cls):
        """운영 환경 추가 검증"""
        super().validate_config()
        
        if cls.SECRET_KEY == 'your_secret_key_change_in_production':
            raise ValueError("SECRET_KEY must be changed in production")

class TestConfig(Config):
    """테스트 환경 설정"""
    DEBUG = True
    TESTING = True
    UPLOAD_FOLDER = 'test_uploads'
    LOG_FOLDER = 'test_logs'
    DELETE_DELAY = 5  # 테스트용 짧은 시간

# 환경별 설정 매핑
config_map = {
    'development': DevelopmentConfig,
    'production': ProductionConfig,
    'testing': TestConfig,
    'default': DevelopmentConfig
}

def get_config(env_name=None):
    """환경에 맞는 설정을 반환합니다."""
    if env_name is None:
        env_name = os.environ.get('FLASK_ENV', 'default')
    
    config_class = config_map.get(env_name, DevelopmentConfig)
    config_class.validate_config()
    
    return config_class