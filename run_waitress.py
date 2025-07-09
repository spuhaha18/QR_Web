from waitress import serve
from app import app  # 여기서 'app'은 Flask 애플리케이션 인스턴스가 정의된 파일 이름입니다

if __name__ == '__main__':
    serve(app, host='0.0.0.0', port=5000)
