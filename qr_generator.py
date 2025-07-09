"""
QR Code generation utilities
"""
import os
import io
import qrcode
import logging
from PIL import Image as PILImage
from cache_manager import cached_qr_code

logger = logging.getLogger(__name__)

class QRCodeGenerator:
    """QR 코드 생성을 담당하는 클래스"""
    
    def __init__(self, box_size=10, border=2, error_correction=qrcode.constants.ERROR_CORRECT_L):
        self.box_size = box_size
        self.border = border
        self.error_correction = error_correction
    
    def create_qr_image(self, qr_text):
        """QR 코드 이미지를 생성하고 PIL Image 객체를 반환합니다."""
        try:
            qr = qrcode.QRCode(
                version=None,
                error_correction=self.error_correction,
                box_size=self.box_size,
                border=self.border
            )
            qr.add_data(qr_text.encode('CP949'))
            qr.make(fit=False)
            img = qr.make_image(fill_color="black", back_color="white")
            
            logger.debug(f"QR code generated for text: {qr_text[:50]}{'...' if len(qr_text) > 50 else ''}")
            return img
        except Exception as e:
            logger.error(f"Failed to generate QR code: {e}")
            raise
    
    def save_qr_to_file(self, qr_text, filepath):
        """QR 코드를 파일로 저장합니다."""
        try:
            qr_img = self.create_qr_image(qr_text)
            qr_img.save(filepath)
            logger.debug(f"QR code saved to: {filepath}")
            return filepath
        except Exception as e:
            logger.error(f"Failed to save QR code to {filepath}: {e}")
            raise
    
    @cached_qr_code(ttl=600)
    def create_qr_base64(self, qr_text):
        """QR 코드를 base64 문자열로 생성합니다."""
        try:
            qr_img = self.create_qr_image(qr_text)
            
            # 메모리에서 이미지 처리
            img_buffer = io.BytesIO()
            qr_img.save(img_buffer, format='PNG')
            img_buffer.seek(0)
            
            import base64
            img_base64 = base64.b64encode(img_buffer.getvalue()).decode('utf-8')
            
            logger.debug(f"QR code converted to base64 (length: {len(img_base64)})")
            return img_base64
        except Exception as e:
            logger.error(f"Failed to create QR code base64: {e}")
            raise
    
    def create_qr_for_excel(self, qr_text, upload_folder, filename):
        """Excel 파일용 QR 코드를 생성하고 파일 경로를 반환합니다."""
        try:
            img_file = os.path.join(upload_folder, f"{filename}.png")
            return self.save_qr_to_file(qr_text, img_file)
        except Exception as e:
            logger.error(f"Failed to create QR code for Excel: {e}")
            raise

# 기본 QR 코드 생성기 인스턴스
default_qr_generator = QRCodeGenerator()