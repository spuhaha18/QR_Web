"""
Cache management utilities for improved performance
"""
import time
import logging
from functools import wraps
from threading import Lock

logger = logging.getLogger(__name__)

class SimpleCache:
    """간단한 메모리 캐시 구현"""
    
    def __init__(self, default_ttl=300):  # 기본 5분 TTL
        self._cache = {}
        self._timestamps = {}
        self._lock = Lock()
        self.default_ttl = default_ttl
    
    def get(self, key):
        """캐시에서 값을 가져옵니다."""
        with self._lock:
            if key not in self._cache:
                return None
            
            # TTL 체크
            if time.time() - self._timestamps[key] > self.default_ttl:
                self._remove_expired(key)
                return None
            
            logger.debug(f"Cache hit for key: {key}")
            return self._cache[key]
    
    def set(self, key, value, ttl=None):
        """캐시에 값을 저장합니다."""
        if ttl is None:
            ttl = self.default_ttl
        
        with self._lock:
            self._cache[key] = value
            self._timestamps[key] = time.time()
            logger.debug(f"Cache set for key: {key}, TTL: {ttl}s")
    
    def _remove_expired(self, key):
        """만료된 키를 제거합니다."""
        if key in self._cache:
            del self._cache[key]
        if key in self._timestamps:
            del self._timestamps[key]
        logger.debug(f"Expired cache entry removed: {key}")
    
    def clear(self):
        """캐시를 모두 지웁니다."""
        with self._lock:
            count = len(self._cache)
            self._cache.clear()
            self._timestamps.clear()
            logger.info(f"Cache cleared, {count} entries removed")
    
    def cleanup_expired(self):
        """만료된 항목들을 정리합니다."""
        current_time = time.time()
        expired_keys = []
        
        with self._lock:
            for key, timestamp in self._timestamps.items():
                if current_time - timestamp > self.default_ttl:
                    expired_keys.append(key)
            
            for key in expired_keys:
                self._remove_expired(key)
        
        if expired_keys:
            logger.info(f"Cleaned up {len(expired_keys)} expired cache entries")
    
    def get_stats(self):
        """캐시 통계를 반환합니다."""
        with self._lock:
            return {
                'total_entries': len(self._cache),
                'memory_usage_estimate': sum(len(str(k)) + len(str(v)) for k, v in self._cache.items()),
                'oldest_entry_age': min(time.time() - ts for ts in self._timestamps.values()) if self._timestamps else 0
            }

# 전역 캐시 인스턴스
qr_cache = SimpleCache(default_ttl=600)  # QR 코드 캐시 (10분)
template_cache = SimpleCache(default_ttl=1800)  # 템플릿 캐시 (30분)

def cached_qr_code(ttl=600):
    """QR 코드 생성 결과를 캐시하는 데코레이터"""
    def decorator(func):
        @wraps(func)
        def wrapper(qr_text, *args, **kwargs):
            # 캐시 키 생성
            cache_key = f"qr_{hash(qr_text)}_{hash(str(args) + str(kwargs))}"
            
            # 캐시에서 확인
            cached_result = qr_cache.get(cache_key)
            if cached_result is not None:
                return cached_result
            
            # 캐시 미스 - 실제 생성
            result = func(qr_text, *args, **kwargs)
            
            # 캐시에 저장 (이미지 객체는 캐시하지 않음 - 메모리 사용량 고려)
            if not hasattr(result, 'save'):  # PIL Image가 아닌 경우만 캐시
                qr_cache.set(cache_key, result, ttl)
            
            return result
        return wrapper
    return decorator

def memory_efficient_file_read(file_path, chunk_size=8192):
    """메모리 효율적인 파일 읽기"""
    try:
        with open(file_path, 'rb') as f:
            while True:
                chunk = f.read(chunk_size)
                if not chunk:
                    break
                yield chunk
    except Exception as e:
        logger.error(f"Error reading file {file_path}: {e}")
        raise

def optimize_memory_usage():
    """메모리 사용량을 최적화합니다."""
    import gc
    
    # 가비지 컬렉션 실행
    collected = gc.collect()
    
    # 캐시 정리
    qr_cache.cleanup_expired()
    template_cache.cleanup_expired()
    
    logger.info(f"Memory optimization completed - GC collected {collected} objects")
    
    return {
        'gc_collected': collected,
        'qr_cache_stats': qr_cache.get_stats(),
        'template_cache_stats': template_cache.get_stats()
    }