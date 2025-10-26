package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *zap.Logger

func Init() {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./logs/app.log", 
		MaxSize:    10,               
		MaxBackups: 3,                
		MaxAge:     28,               
		Compress:   true,             
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		jst := time.FixedZone("Asia/Tokyo", 9*60*60)
		enc.AppendString(t.In(jst).Format(time.RFC3339))
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(lumberJackLogger),
		zap.InfoLevel,
	)

	log = zap.New(core)
	defer log.Sync()
}

func Info(message string, fields ...zap.Field) {
	log.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	log.Debug(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	log.Error(message, fields...)
}
