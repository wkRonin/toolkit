package logger

import (
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Mode       int    `yaml:"mode"`
	Level      string `yaml:"level"`
	Filename   string `yaml:"filename"`
	MaxSize    int    `yaml:"maxSize"`
	MaxBackups int    `yaml:"maxBackups"`
	MaxAge     int    `yaml:"maxAge"`
}

func (c *Config) BuildConfig() *zap.Logger {
	// 创建Core三大件，进行初始化
	writeSyncer := c.getLogWriter()
	encoder := c.getEncoder()
	var level = new(zapcore.Level)
	err := level.UnmarshalText([]byte(c.Level))
	if err != nil {
		// 默认为info
		_ = level.Set("info")
	}
	// 创建核心-->如果是1模式，就在控制台和文件都打印，否则就只写到文件中
	var core zapcore.Core
	if c.Mode == 1 {
		// 开发模式，日志输出到终端
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		// NewTee创建一个核心，将日志条目复制到两个或多个底层核心中。
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, writeSyncer, level),
			zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), level),
		)
	} else {
		core = zapcore.NewCore(encoder, writeSyncer, level)
	}
	// 创建 logger 对象
	l := zap.New(core, zap.AddCaller())
	// zap底层 API 可以设置缓存，所以一般使用defer logger.Sync()将缓存同步到文件中。
	defer l.Sync()
	return l
}

// 获取切割的文件，给初始化logger使用的
func (c *Config) getLogWriter() zapcore.WriteSyncer {
	// 使用 lumberjack 归档切片日志
	lumberJackLogger := &lumberjack.Logger{
		Filename:   c.Filename,
		MaxSize:    c.MaxSize,
		MaxBackups: c.MaxBackups,
		MaxAge:     c.MaxAge,
	}
	return zapcore.AddSync(lumberJackLogger)
}

// 获取Encoder，给初始化logger使用的
func (c *Config) getEncoder() zapcore.Encoder {
	// 使用zap提供的 NewProductionEncoderConfig
	encoderConfig := zap.NewProductionEncoderConfig()
	// 设置时间格式
	// encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	// 时间的key
	encoderConfig.TimeKey = "time"
	// 级别
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	// 显示调用者信息
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	// 返回json 格式的 日志编辑器
	return zapcore.NewJSONEncoder(encoderConfig)
}
