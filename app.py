"""
Optimized Flask application for QR Web Label Generator
"""
import logging
import os
import io
import base64
import time
from datetime import datetime
from flask import Flask, render_template, request, send_file, redirect, url_for, flash, jsonify, make_response

# 로컬 모듈
from utils import (
    get_client_info, log_client_access, create_directory_if_not_exists,
    delete_file_later, validate_and_clean_input, safe_int_conversion
)
from qr_generator import default_qr_generator
from excel_generator import ExcelLabelGenerator
from performance_monitor import monitor_performance, performance_monitor, request_metrics, get_system_metrics
from cache_manager import optimize_memory_usage

# 설정 로드
from config import get_config
config = get_config()

# Flask 앱 설정
app = Flask(__name__)
app.config.from_object(config)
app.secret_key = config.SECRET_KEY

# 상수 정의
UPLOAD_FOLDER = config.UPLOAD_FOLDER
LOG_FOLDER = config.LOG_FOLDER
DELETE_DELAY = config.DELETE_DELAY

# 디렉토리 생성
create_directory_if_not_exists(UPLOAD_FOLDER)
create_directory_if_not_exists(LOG_FOLDER)

# 로깅 설정
log_file_path = os.path.join(LOG_FOLDER, 'app.log')
logging.basicConfig(
    filename=log_file_path,
    level=getattr(logging, config.LOG_LEVEL),
    format=config.LOG_FORMAT
)
logger = logging.getLogger(__name__)

# Excel 생성기 인스턴스
excel_generator = ExcelLabelGenerator(UPLOAD_FOLDER)

# 에러 처리 데코레이터
def handle_errors(f):
    """에러 처리를 위한 데코레이터"""
    def wrapper(*args, **kwargs):
        try:
            return f(*args, **kwargs)
        except Exception as e:
            client_ip, _ = get_client_info()
            logger.error(f"Error in {f.__name__} for client {client_ip}: {str(e)}", exc_info=True)
            
            if request.is_json:
                return jsonify({'error': '서버 오류가 발생했습니다.'}), 500
            else:
                flash("서버 오류가 발생했습니다.", "error")
                return redirect(url_for('index'))
    
    wrapper.__name__ = f.__name__
    return wrapper

# 데이터 검증 함수들
def validate_document_type(doc_type):
    """문서 타입 검증"""
    return doc_type in ['1', '2']

def validate_binder_size(binder_size, doc_type):
    """바인더 크기 검증"""
    valid_sizes = [1, 3, 5, 7]
    if binder_size not in valid_sizes:
        return False
    
    # 과제 문서는 3cm 미만 불가
    if doc_type == '2' and binder_size == 1:
        return False
    
    return True

def validate_required_fields(form_data, doc_type):
    """필수 필드 검증"""
    if doc_type == '1':
        required_fields = ['eq_number', 'eq_doc_number', 'eq_doc_title', 
                          'eq_doc_count', 'eq_doc_department', 'eq_doc_year']
    else:
        required_fields = ['pjt_number', 'pjt_test_number', 'pjt_doc_title', 
                          'pjt_doc_writer', 'pjt_doc_count']
    
    for field in required_fields:
        if not form_data.get(field):
            return False, field
    
    return True, None

def process_form_data(form_data, doc_type):
    """폼 데이터를 처리하고 정리합니다."""
    client_ip, _ = get_client_info()
    
    if doc_type == '1':
        data = {
            'eq_number': validate_and_clean_input(form_data.get('eq_number')),
            'eq_doc_number': validate_and_clean_input(form_data.get('eq_doc_number')),
            'eq_doc_title': validate_and_clean_input(form_data.get('eq_doc_title')),
            'eq_doc_count': safe_int_conversion(form_data.get('eq_doc_count')),
            'eq_doc_department': validate_and_clean_input(form_data.get('eq_doc_department')),
            'eq_doc_year': safe_int_conversion(form_data.get('eq_doc_year'), datetime.now().year)
        }
        
        logger.info(f"Equipment document data - Number: {data['eq_number']}, "
                   f"Doc Number: {data['eq_doc_number']}, "
                   f"Title: {data['eq_doc_title'][:50]}{'...' if len(data['eq_doc_title']) > 50 else ''}, "
                   f"Count: {data['eq_doc_count']}, Department: {data['eq_doc_department']}, "
                   f"Year: {data['eq_doc_year']}")
    else:
        data = {
            'pjt_number': validate_and_clean_input(form_data.get('pjt_number')),
            'pjt_test_number': validate_and_clean_input(form_data.get('pjt_test_number')),
            'pjt_doc_title': validate_and_clean_input(form_data.get('pjt_doc_title')),
            'pjt_doc_writer': validate_and_clean_input(form_data.get('pjt_doc_writer')),
            'pjt_doc_count': safe_int_conversion(form_data.get('pjt_doc_count'))
        }
        
        logger.info(f"Project document data - Project: {data['pjt_number']}, "
                   f"Test Number: {data['pjt_test_number']}, "
                   f"Title: {data['pjt_doc_title'][:50]}{'...' if len(data['pjt_doc_title']) > 50 else ''}, "
                   f"Writer: {data['pjt_doc_writer']}, Count: {data['pjt_doc_count']}")
    
    return data

# 라우트 정의
@app.route('/')
def index():
    """메인 페이지"""
    log_client_access("Index page")
    current_year = datetime.now().year
    return render_template('index.html', current_year=current_year)

@app.route('/create_label', methods=['POST'])
@handle_errors
@monitor_performance("web_label_creation")
def create_label():
    """라벨 생성 (웹 인터페이스)"""
    client_ip, user_agent = get_client_info()
    logger.info(f"Create label request received from {client_ip}")
    
    # 기본 데이터 검증
    doc_type = request.form.get('doc_type')
    try:
        binder_size = int(request.form.get('binder_size'))
    except (ValueError, TypeError):
        flash("잘못된 바인더 크기입니다.", "error")
        logger.warning(f"Invalid binder size from client {client_ip}")
        return redirect(url_for('index'))
    
    logger.info(f"Request details - Document type: {doc_type}, Binder size: {binder_size}cm")
    
    # 문서 타입 검증
    if not validate_document_type(doc_type):
        flash("잘못된 문서 종류입니다.", "error")
        logger.warning(f"Invalid document type: {doc_type} from client {client_ip}")
        return redirect(url_for('index'))
    
    # 바인더 크기 검증
    if not validate_binder_size(binder_size, doc_type):
        flash("과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.", "error")
        logger.warning(f"Invalid binder size selected - Client: {client_ip}, "
                      f"Doc type: {doc_type}, Binder size: {binder_size}cm")
        return redirect(url_for('index'))
    
    # 필수 필드 검증
    is_valid, missing_field = validate_required_fields(request.form, doc_type)
    if not is_valid:
        flash("모든 필드를 채워주세요.", "error")
        logger.warning(f"Missing required field: {missing_field} - Client: {client_ip}")
        return redirect(url_for('index'))
    
    # 데이터 처리
    data = process_form_data(request.form, doc_type)
    
    # 라벨 생성
    logger.info(f"Starting label generation process for client {client_ip}")
    filepath, filename = excel_generator.create_label_excel(doc_type, binder_size, data)
    
    # 파일 정보 로깅
    from utils import get_file_size_safe
    file_size = get_file_size_safe(filepath)
    logger.info(f"Label generation completed - File: {filename}, Size: {file_size} bytes")
    
    # 파일 삭제 예약
    delete_file_later(filepath, DELETE_DELAY)
    logger.info(f"File deletion scheduled for {filename} ({DELETE_DELAY} seconds delay)")
    
    # 다운로드 응답
    response = make_response(send_file(filepath, as_attachment=True, download_name=filename))
    response.set_cookie('download_complete', 'true', max_age=10)
    logger.info(f"File download initiated for client {client_ip} - File: {filename}")
    
    return response

@app.route('/api/create_label', methods=['POST'])
@handle_errors
@monitor_performance("api_label_creation")
def api_create_label():
    """라벨 생성 API 엔드포인트"""
    client_ip, _ = get_client_info()
    data = request.get_json()
    
    if not data:
        return jsonify({'error': '잘못된 JSON 데이터입니다.'}), 400
    
    doc_type = data.get('doc_type')
    binder_size = data.get('binder_size')
    
    # 데이터 유효성 검사
    if not validate_document_type(doc_type):
        return jsonify({'error': '잘못된 문서 종류입니다.'}), 400
    
    if not validate_binder_size(binder_size, doc_type):
        return jsonify({'error': '과제 문서의 경우 3cm 미만 바인더 크기를 선택할 수 없습니다.'}), 400
    
    # 필수 필드 검사
    is_valid, missing_field = validate_required_fields(data, doc_type)
    if not is_valid:
        return jsonify({'error': f'필수 필드가 누락되었습니다: {missing_field}'}), 400
    
    # 라벨 생성
    processed_data = process_form_data(data, doc_type)
    filepath, filename = excel_generator.create_label_excel(doc_type, binder_size, processed_data)
    
    # 파일 삭제 예약
    delete_file_later(filepath, DELETE_DELAY)
    
    return jsonify({
        'success': True,
        'message': '라벨이 성공적으로 생성되었습니다.',
        'filename': filename,
        'download_url': f'/download/{filename}'
    })

@app.route('/api/qr_image/<path:qr_text>', methods=['GET'])
@handle_errors
def api_qr_image(qr_text):
    """QR 코드 이미지 생성 (PNG)"""
    if not qr_text:
        return jsonify({'error': 'QR 코드 텍스트가 제공되지 않았습니다.'}), 400
    
    qr_img = default_qr_generator.create_qr_image(qr_text)
    
    img_io = io.BytesIO()
    qr_img.save(img_io, 'PNG')
    img_io.seek(0)
    
    return send_file(img_io, mimetype='image/png', as_attachment=False, download_name='qr_code.png')

@app.route('/api/qr_image_base64', methods=['POST'])
@handle_errors
def api_qr_image_base64():
    """QR 코드 이미지 생성 (Base64)"""
    data = request.get_json()
    if not data or not data.get('text'):
        return jsonify({'error': 'QR 코드 텍스트가 제공되지 않았습니다.'}), 400
    
    qr_text = data['text']
    img_base64 = default_qr_generator.create_qr_base64(qr_text)
    
    return jsonify({
        'success': True,
        'image_base64': img_base64,
        'mime_type': 'image/png'
    })

@app.route('/download/<filename>', methods=['GET'])
@handle_errors
def download_file(filename):
    """파일 다운로드"""
    filepath = os.path.join(UPLOAD_FOLDER, filename)
    if os.path.exists(filepath):
        return send_file(filepath, as_attachment=True, download_name=filename)
    else:
        return jsonify({'error': '파일을 찾을 수 없습니다.'}), 404

@app.route('/logs')
def logs_page():
    """로그 확인 페이지"""
    log_client_access("Logs page")
    return render_template('logs.html')

@app.route('/api/logs', methods=['GET'])
@handle_errors
def api_get_logs():
    """로그 조회 API"""
    client_ip, _ = get_client_info()
    
    # 쿼리 파라미터 처리
    lines = min(request.args.get('lines', 100, type=int), 1000)
    level = request.args.get('level', 'all').upper()
    search = request.args.get('search', '')
    
    logger.info(f"API logs request from {client_ip} - lines: {lines}, level: {level}, search: '{search}'")
    
    if not os.path.exists(log_file_path):
        return jsonify({
            'success': True,
            'logs': [],
            'message': '로그 파일이 아직 생성되지 않았습니다.'
        })
    
    logs = []
    with open(log_file_path, 'r', encoding='utf-8') as f:
        all_lines = f.readlines()
    
    # 최근 줄부터 가져오기
    recent_lines = all_lines[-lines:] if len(all_lines) > lines else all_lines
    
    for line in recent_lines:
        line = line.strip()
        if not line:
            continue
        
        # 레벨 필터링
        if level != 'ALL' and level not in line:
            continue
        
        # 검색어 필터링
        if search and search.lower() not in line.lower():
            continue
        
        logs.append(line)
    
    return jsonify({
        'success': True,
        'logs': logs,
        'total_lines': len(logs),
        'requested_lines': lines,
        'level_filter': level,
        'search_filter': search
    })

@app.route('/api/logs/clear', methods=['POST'])
@handle_errors
def api_clear_logs():
    """로그 파일 초기화"""
    if os.path.exists(log_file_path):
        # 로그 파일 백업
        backup_path = f"app_backup_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log"
        try:
            with open(log_file_path, 'r', encoding='utf-8') as src:
                with open(backup_path, 'w', encoding='utf-8') as dst:
                    dst.write(src.read())
            logger.info(f"Log file backed up to: {backup_path}")
        except Exception as backup_error:
            logger.warning(f"Failed to backup log file: {backup_error}")
        
        # 로그 파일 초기화
        open(log_file_path, 'w').close()
        logger.info("Log file cleared by user request")
        
        return jsonify({
            'success': True,
            'message': '로그 파일이 초기화되었습니다.',
            'backup_file': backup_path if 'backup_path' in locals() else None
        })
    else:
        return jsonify({
            'success': True,
            'message': '초기화할 로그 파일이 없습니다.'
        })

@app.route('/api/logs/download', methods=['GET'])
@handle_errors
def api_download_logs():
    """로그 파일 다운로드"""
    if not os.path.exists(log_file_path):
        return jsonify({'error': '다운로드할 로그 파일이 없습니다.'}), 404
    
    download_name = f"app_logs_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log"
    return send_file(log_file_path, as_attachment=True, download_name=download_name, mimetype='text/plain')

@app.route('/api/health', methods=['GET'])
def api_health():
    """서비스 상태 확인"""
    return jsonify({
        'status': 'healthy',
        'timestamp': datetime.now().isoformat(),
        'version': '1.0.0'
    })

@app.route('/api/docs')
def api_docs():
    """API 문서 페이지"""
    log_client_access("API documentation page")
    return render_template('api_docs.html')

@app.route('/api/performance', methods=['GET'])
def api_performance():
    """성능 통계 API"""
    try:
        stats = performance_monitor.get_stats()
        slow_ops = performance_monitor.get_slow_operations(threshold=0.5)
        system_metrics = get_system_metrics()
        request_stats = request_metrics.get_request_stats()
        
        return jsonify({
            'success': True,
            'performance_stats': stats,
            'slow_operations': slow_ops,
            'system_metrics': system_metrics,
            'request_stats': request_stats
        })
    except Exception as e:
        logger.error(f"Error getting performance stats: {e}")
        return jsonify({'error': '성능 통계 조회 중 오류가 발생했습니다.'}), 500

@app.route('/api/system/optimize', methods=['POST'])
@handle_errors
def api_optimize_system():
    """시스템 최적화 API"""
    optimization_result = optimize_memory_usage()
    
    return jsonify({
        'success': True,
        'message': '시스템 최적화가 완료되었습니다.',
        'optimization_result': optimization_result
    })

@app.route('/api/system/status', methods=['GET'])
def api_system_status():
    """시스템 상태 확인 API (확장된 버전)"""
    try:
        system_metrics = get_system_metrics()
        performance_stats = performance_monitor.get_stats()
        request_stats = request_metrics.get_request_stats()
        
        # 간단한 헬스체크
        health_status = 'healthy'
        issues = []
        
        # 메모리 사용량 체크
        if system_metrics.get('memory_percent', 0) > 80:
            health_status = 'warning'
            issues.append('High memory usage')
        
        # 에러율 체크
        for name, stats in performance_stats.items():
            if stats.get('error_rate', 0) > 0.1:  # 10% 이상 에러율
                health_status = 'warning'
                issues.append(f'High error rate in {name}')
        
        return jsonify({
            'status': health_status,
            'timestamp': datetime.now().isoformat(),
            'version': '2.0.0',
            'system_metrics': system_metrics,
            'request_stats': request_stats,
            'issues': issues
        })
    except Exception as e:
        logger.error(f"Error getting system status: {e}")
        return jsonify({
            'status': 'error',
            'timestamp': datetime.now().isoformat(),
            'error': str(e)
        }), 500

# 요청 추적 미들웨어
@app.before_request
def before_request():
    """요청 시작 시 호출"""
    if request.endpoint and not request.endpoint.startswith('static'):
        client_ip, _ = get_client_info()
        request_id = f"{client_ip}_{int(time.time() * 1000000)}"
        request.request_id = request_id
        request_metrics.start_request(request_id, request.endpoint, client_ip)

@app.after_request
def after_request(response):
    """요청 완료 시 호출"""
    if hasattr(request, 'request_id'):
        error = None if response.status_code < 400 else f"HTTP {response.status_code}"
        request_metrics.end_request(request.request_id, response.status_code, error)
    return response

if __name__ == '__main__':
    logger.info(f"Starting QR Web application with {config.__class__.__name__}")
    app.run(host=config.HOST, port=config.PORT, debug=config.DEBUG)