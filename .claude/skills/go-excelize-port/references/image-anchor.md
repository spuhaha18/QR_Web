# QR 이미지 임베드: add_image → AddPictureFromBytes

## openpyxl 현재 동작
```python
img_obj = Image(img_file)
img_obj.width = 75; img_obj.height = 75   # 절대 75x75 px 강제
sheet.add_image(img_obj, config['cell_pos'])  # 셀 좌상단 one-cell 앵커
```
원본 QR PNG 픽셀 크기와 무관하게 **75×75px 박스**로 강제, 앵커 셀(예: E9) 좌상단 고정.

## excelize 재현
excelize는 절대 px가 아니라 **스케일 팩터**로 크기를 정한다. 원본 px를 디코드해 스케일 계산:
```go
cfg, _, err := image.DecodeConfig(bytes.NewReader(pngBytes)) // image/png import 필요
scaleX := 75.0 / float64(cfg.Width)
scaleY := 75.0 / float64(cfg.Height)

err = f.AddPictureFromBytes(sheet, cellPos, &excelize.Picture{
    Extension: ".png",
    File:      pngBytes,
    Format: &excelize.GraphicOptions{
        ScaleX:      scaleX,
        ScaleY:      scaleY,
        OffsetX:     0,
        OffsetY:     0,
        Positioning: "oneCell", // 셀 좌상단 절대 앵커 (openpyxl OneCellAnchor 대응)
        AutoFit:     false,
        LockAspectRatio: false,
    },
})
```
> excelize 버전별 API 시그니처 확인(`AddPictureFromBytes`의 인자 형태가 버전마다 다름 — 일부는 `(sheet, cell, name, extension, []byte, *GraphicOptions)`). 설치된 버전 godoc 확인 후 맞춘다.

## 검증 포인트 (parity 위험 #2)
- QR이 **같은 셀**(바인더별 E9/D9/D8/B9 등)에 앉는가.
- 화면상 크기가 ~75px로 동일한가 — LibreOffice/Excel로 육안 확인.
- 앵커가 two-cell이면 셀 리사이즈 시 이미지가 따라 움직임 → one-cell 절대 앵커 확인.
- paste 모드: 클라이언트 업로드 PNG는 임의 크기 → 반드시 디코드 후 스케일. auto 모드: go-qrcode 출력 px 고정이라 스케일 상수화 가능.

## 순서 주의
현재 코드는 모든 시트를 만든 **뒤** QR 루프를 전체 시트에 돌린다. CopySheet가 이미지를 복제하지 않아도 되도록 **시트 복제 후 시트별로 임베드** 순서를 지킨다.
