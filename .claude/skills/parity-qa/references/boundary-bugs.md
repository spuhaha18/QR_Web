# 경계면 버그 패턴 체크리스트

API 백엔드와 프론트 사이 경계에서 자주 나는 버그. 핸들러 응답과 fetch 파싱을 **동시에** 읽고 대조한다.

## 필드명 불일치
- [ ] 백엔드 FormData 키(snake_case `doc_type`, `binder_size`, `qr_order`, `qr_images`) vs 프론트 append 키 일치?
- [ ] JSON 응답 키(`image_base64`, `mime_type`, `status`) vs 프론트 구조분해 일치?
- [ ] camelCase/snake_case 혼동 없음?

## 응답 형식 불일치 (마이그레이션 핵심)
- [ ] `/api/create_label`이 스트리밍으로 변경(`download_url`→바이트) → 프론트가 여전히 `download_url` 기대하지 않는가?
- [ ] `.xlsx` 바이너리 응답을 프론트 `res.blob()`으로 받는가? (`res.json()` 아님)
- [ ] 에러는 `{error: "..."}` JSON 4xx인가? 프론트가 `res.ok` 체크 후 `res.json().error` 파싱하는가?

## 다운로드 흐름
- [ ] Content-Disposition `filename` 파싱 정규식이 백엔드 헤더 형식과 맞는가?
- [ ] Content-Type `application/vnd.openxml...sheet` 설정됨?

## 검증 규칙 일치
- [ ] qr_order 순열 계약: 프론트 전송 형식 vs 백엔드 재정렬 기대 일치?
- [ ] qr_images 개수 == doc_count를 프론트도 제출 전 체크(UX) + 백엔드 강제(보안)?
- [ ] 파일 크기 ≤2MB, PNG 검증을 양쪽에서?

## 상태코드/에러 UX
- [ ] 400 한국어 에러가 Toast에 그대로 표시되는가?
- [ ] 500 시 일반 메시지("서버 오류가 발생했습니다.")?

## 인코딩
- [ ] 한글 필드가 multipart에서 UTF-8로 전송되고 백엔드가 UTF-8로 파싱하는가? (QR 페이로드 CP949는 별개 — 서버 내부 인코딩)
