# Config: config.py → Go

## Go 구현 (internal/config/config.go)
stdlib만. viper 불필요.
```go
package config

type Config struct {
    Host             string
    Port             int
    LogLevel         string
    LogFile          string
    MaxContentLength int64 // 16MB
    MaxQRFiles       int   // 50
    MaxQRFileSize    int64 // 2MB
    QRBoxSize        int   // 10
    QRBorder         int   // 2
    Version          string
}

func Load() *Config {
    return &Config{
        Host:             getEnv("HOST", "0.0.0.0"),
        Port:             getEnvInt("PORT", 5000),
        LogLevel:         getEnv("LOG_LEVEL", "INFO"),
        MaxContentLength: getEnvInt64("MAX_CONTENT_LENGTH", 16*1024*1024),
        MaxQRFiles:       getEnvInt("MAX_QR_FILES", 50),
        MaxQRFileSize:    getEnvInt64("MAX_QR_FILE_SIZE", 2*1024*1024),
        QRBoxSize:        getEnvInt("QR_BOX_SIZE", 10),
        QRBorder:         getEnvInt("QR_BORDER", 2),
    }
}
// getEnv/getEnvInt/getEnvInt64/getEnvBool 헬퍼: os.Getenv + strconv + 기본값
```

## env 매핑
| config.py | Go env | 기본 |
|-----------|--------|------|
| FLASK_HOST | HOST | 0.0.0.0 |
| FLASK_PORT | PORT | 5000 |
| MAX_CONTENT_LENGTH | MAX_CONTENT_LENGTH | 16MB |
| MAX_QR_FILES | MAX_QR_FILES | 50 |
| MAX_QR_FILE_SIZE | MAX_QR_FILE_SIZE | 2MB |
| LOG_LEVEL | LOG_LEVEL | INFO |
| QR_BOX_SIZE | QR_BOX_SIZE | 10 |
| QR_BORDER | QR_BORDER | 2 |

## 드롭 (스트리밍/SPA 채택 시 불필요)
- `SECRET_KEY` (Flask 세션/flash 없음)
- `DELETE_DELAY`, `UPLOAD_FOLDER`, `LOG_FOLDER`(스트리밍 시 임시파일 없음)
- `QR_CACHE_TTL`, 성능 모니터링 플래그

## 선택
- `.env` 로더 유지 원하면 `github.com/joho/godotenv` (기존 `.env` 호환).
- dev/prod/test 클래스 계층 불필요 — env + `Validate()`로 통합.
