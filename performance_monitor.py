"""
Performance monitoring utilities
"""
import time
import threading
import logging
from functools import wraps
from collections import defaultdict, deque
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)

class PerformanceMonitor:
    """성능 모니터링 클래스"""
    
    def __init__(self, max_history=1000):
        self.max_history = max_history
        self.metrics = defaultdict(lambda: {
            'count': 0,
            'total_time': 0,
            'min_time': float('inf'),
            'max_time': 0,
            'recent_times': deque(maxlen=100),  # 최근 100회
            'errors': 0
        })
        self.lock = threading.Lock()
    
    def record_metric(self, name, execution_time, error=False):
        """메트릭을 기록합니다."""
        with self.lock:
            metric = self.metrics[name]
            metric['count'] += 1
            metric['total_time'] += execution_time
            metric['min_time'] = min(metric['min_time'], execution_time)
            metric['max_time'] = max(metric['max_time'], execution_time)
            metric['recent_times'].append(execution_time)
            
            if error:
                metric['errors'] += 1
            
            logger.debug(f"Recorded metric {name}: {execution_time:.3f}s (error: {error})")
    
    def get_stats(self, name=None):
        """통계를 반환합니다."""
        with self.lock:
            if name:
                return self._calculate_stats(name, self.metrics[name])
            
            return {name: self._calculate_stats(name, metric) 
                   for name, metric in self.metrics.items()}
    
    def _calculate_stats(self, name, metric):
        """개별 메트릭의 통계를 계산합니다."""
        if metric['count'] == 0:
            return {'count': 0}
        
        avg_time = metric['total_time'] / metric['count']
        recent_times = list(metric['recent_times'])
        recent_avg = sum(recent_times) / len(recent_times) if recent_times else 0
        
        return {
            'name': name,
            'count': metric['count'],
            'total_time': metric['total_time'],
            'avg_time': avg_time,
            'min_time': metric['min_time'],
            'max_time': metric['max_time'],
            'recent_avg_time': recent_avg,
            'error_rate': metric['errors'] / metric['count'],
            'recent_count': len(recent_times)
        }
    
    def get_slow_operations(self, threshold=1.0):
        """느린 작업들을 반환합니다."""
        slow_ops = []
        
        with self.lock:
            for name, metric in self.metrics.items():
                if metric['count'] > 0:
                    avg_time = metric['total_time'] / metric['count']
                    if avg_time > threshold:
                        slow_ops.append({
                            'name': name,
                            'avg_time': avg_time,
                            'max_time': metric['max_time'],
                            'count': metric['count']
                        })
        
        return sorted(slow_ops, key=lambda x: x['avg_time'], reverse=True)
    
    def reset_metrics(self):
        """모든 메트릭을 초기화합니다."""
        with self.lock:
            self.metrics.clear()
            logger.info("Performance metrics reset")

# 전역 모니터 인스턴스
performance_monitor = PerformanceMonitor()

def monitor_performance(operation_name=None):
    """성능 모니터링 데코레이터"""
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            name = operation_name or f"{func.__module__}.{func.__name__}"
            start_time = time.time()
            error_occurred = False
            
            try:
                result = func(*args, **kwargs)
                return result
            except Exception as e:
                error_occurred = True
                raise
            finally:
                execution_time = time.time() - start_time
                performance_monitor.record_metric(name, execution_time, error_occurred)
        
        return wrapper
    return decorator

def get_system_metrics():
    """시스템 메트릭을 반환합니다."""
    import psutil
    import os
    
    try:
        # 현재 프로세스 정보
        process = psutil.Process(os.getpid())
        
        # 메모리 사용량
        memory_info = process.memory_info()
        
        # CPU 사용량
        cpu_percent = process.cpu_percent()
        
        # 파일 디스크립터 수
        try:
            num_fds = process.num_fds()
        except AttributeError:
            num_fds = 0  # Windows에서는 지원하지 않음
        
        return {
            'memory_rss': memory_info.rss,
            'memory_vms': memory_info.vms,
            'memory_percent': process.memory_percent(),
            'cpu_percent': cpu_percent,
            'num_threads': process.num_threads(),
            'num_fds': num_fds,
            'create_time': process.create_time()
        }
    except ImportError:
        logger.warning("psutil not available, system metrics disabled")
        return {}
    except Exception as e:
        logger.error(f"Error getting system metrics: {e}")
        return {}

class RequestMetrics:
    """요청별 메트릭 추적"""
    
    def __init__(self):
        self.active_requests = {}
        self.completed_requests = deque(maxlen=1000)
        self.lock = threading.Lock()
    
    def start_request(self, request_id, endpoint, client_ip):
        """요청 시작을 기록합니다."""
        with self.lock:
            self.active_requests[request_id] = {
                'endpoint': endpoint,
                'client_ip': client_ip,
                'start_time': time.time(),
                'timestamp': datetime.now()
            }
    
    def end_request(self, request_id, status_code=200, error=None):
        """요청 완료를 기록합니다."""
        with self.lock:
            if request_id not in self.active_requests:
                return
            
            request_info = self.active_requests.pop(request_id)
            duration = time.time() - request_info['start_time']
            
            completed_request = {
                **request_info,
                'duration': duration,
                'status_code': status_code,
                'error': error,
                'end_time': time.time()
            }
            
            self.completed_requests.append(completed_request)
    
    def get_active_requests(self):
        """활성 요청 목록을 반환합니다."""
        with self.lock:
            return list(self.active_requests.values())
    
    def get_recent_requests(self, limit=100):
        """최근 완료된 요청들을 반환합니다."""
        with self.lock:
            return list(self.completed_requests)[-limit:]
    
    def get_request_stats(self):
        """요청 통계를 반환합니다."""
        with self.lock:
            recent = list(self.completed_requests)
            
            if not recent:
                return {'total_requests': 0}
            
            total_requests = len(recent)
            total_duration = sum(r['duration'] for r in recent)
            avg_duration = total_duration / total_requests
            
            # 상태 코드별 통계
            status_counts = defaultdict(int)
            for r in recent:
                status_counts[r['status_code']] += 1
            
            # 엔드포인트별 통계
            endpoint_counts = defaultdict(int)
            for r in recent:
                endpoint_counts[r['endpoint']] += 1
            
            return {
                'total_requests': total_requests,
                'avg_duration': avg_duration,
                'active_requests': len(self.active_requests),
                'status_codes': dict(status_counts),
                'endpoints': dict(endpoint_counts)
            }

# 전역 요청 메트릭 인스턴스
request_metrics = RequestMetrics()