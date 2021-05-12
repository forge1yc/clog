package clog

import (
	"fmt"
	"log"
	"math/rand"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	LEVEL_FLAGS = [...]string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL",
		"PUBLIC"}
)

const (
	TRACE = iota
	DEBUG
	INFO
	WARNING
	ERROR
	FATAL
	PUBLIC
)
func init() {
	logger_default = NewLogger()
}

const tunnel_size_default = 1024
const buffer_bytes_cnt = 1024

type Record struct {
	time   time.Time
	code   string
	line   int
	info   string
	level  int
	args   []interface{}
	hight  bool
	fields []Field
}

type textEncoder struct {
	bytes []byte
	level int
}

func (enc *textEncoder) truncate() {
	enc.bytes = enc.bytes[:0]
}

func (enc *textEncoder) Write(p []byte) (int, error) {
	enc.bytes = append(enc.bytes, p...)
	return len(p), nil
}

var textPool = sync.Pool{New: func() interface{} {
	return &textEncoder{
		bytes: make([]byte, 0, buffer_bytes_cnt),
	}
}}

func (r *Record) Bytes(enc *textEncoder) {
	enc.bytes = append(enc.bytes, '[')
	enc.bytes = r.time.AppendFormat(enc.bytes, logger_default.layout)
	enc.bytes = append(enc.bytes, "] ["...)
	enc.bytes = append(enc.bytes, LEVEL_FLAGS[r.level]...)
	enc.bytes = append(enc.bytes, "] ["...)
	enc.bytes = append(enc.bytes, r.code...)
	enc.bytes = append(enc.bytes, ':')
	enc.bytes = strconv.AppendInt(enc.bytes, int64(r.line), 10)
	enc.bytes = append(enc.bytes, "] "...)
	meta := " "
	if r.level == PUBLIC {
		meta = "||"
	}
	if r.hight {
		enc.bytes = append(enc.bytes, r.info...)
		for _, field := range r.fields {
			enc.bytes = append(enc.bytes, meta...)
			enc.bytes = append(enc.bytes, field.key...)
			enc.bytes = append(enc.bytes, byte('='))
			enc.bytes = field.WriteValue(enc.bytes)
		}
	} else {
		if len(r.args) == 0 {
			enc.bytes = append(enc.bytes, r.info...)
		} else {
			fmt.Fprintf(enc, r.info, r.args...)
		}
	}
	enc.bytes = append(enc.bytes, '\n')
}

type Writer interface {
	Init() error
	Write(*textEncoder) error
}

type Rotater interface {
	Rotate() error
	SetPathPattern(string) error
}

type Deleter interface {
	Delete() error
}

type Flusher interface {
	Flush() error
}

type Logger struct {
	writers []Writer
	tunnel  chan *textEncoder
	level   int
	c       chan bool
	layout  string
}

func NewLogger() *Logger {
	l := new(Logger)
	l.writers = make([]Writer, 0, 2)
	l.tunnel = make(chan *textEncoder, tunnel_size_default)
	l.c = make(chan bool, 1)
	l.level = DEBUG
	l.layout = "2006-01-02T15:04:05.000+0800"

	go boostrapLogWriter(l)

	return l
}

func (l *Logger) Register(w Writer) {
	if err := w.Init(); err != nil {
		panic(err)
	}
	l.writers = append(l.writers, w)
}

func (l *Logger) SetLevel(lvl int) {
	l.level = lvl
}

func (l *Logger) SetLayout(layout string) {
	l.layout = layout
}

func (l *Logger) Public(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(PUBLIC, fmt, args...)
}

func (l *Logger) Trace(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(TRACE, fmt, args...)
}

func (l *Logger) Debug(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(DEBUG, fmt, args...)
}

func (l *Logger) Warn(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(WARNING, fmt, args...)
}

func (l *Logger) Info(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(INFO, fmt, args...)
}

func (l *Logger) Error(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(ERROR, fmt, args...)
}

func (l *Logger) Fatal(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(FATAL, fmt, args...)
}

func (l *Logger) close() {
	close(l.tunnel)
	<-l.c

	for _, w := range l.writers {
		if f, ok := w.(Flusher); ok {
			if err := f.Flush(); err != nil {
				log.Println(err)
			}
		}
	}

}

func (l *Logger) deliverRecordToWriter(level int, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	// source code, file and line num
	_, file, line, ok := runtime.Caller(2)
	r := &Record{}
	r.hight = false
	r.info = format
	if ok {
		r.code = path.Base(file)
		r.line = line
	}
	r.time = time.Now()
	r.level = level
	r.args = args

	enc := textPool.Get().(*textEncoder)
	enc.truncate() // 为啥这里需要truncate
	enc.level = r.level
	r.Bytes(enc)

	l.tunnel <- enc
}

func (l *Logger) deliverRecordToWriterHight(level int, with string,
	fields ...Field) {
	if level < l.level {
		return
	}
	// source code,file and line num
	_, file, line, ok := runtime.Caller(2)
	r := &Record{}
	r.hight = true
	r.info = with
	if ok {
		r.code = path.Base(file)
		r.line = line
	}
	r.time = time.Now()
	r.level = level
	r.fields = fields

	enc := textPool.Get().(*textEncoder)
	enc.truncate()
	enc.level = r.level
	r.Bytes(enc)

	l.tunnel <- enc
}

func boostrapLogWriter(logger *Logger) {
	if logger == nil {
		panic("logger is nil")
	}

	var (
		enc *textEncoder
		ok  bool
	)

	if enc, ok = <-logger.tunnel; !ok {
		logger.c <- true
		return
	}

	for _, w := range logger.writers {
		if err := w.Write(enc); err != nil {
			log.Println(err)
		}
	}

	flushTimer := time.NewTimer(time.Millisecond * 500)
	rotateTimer := time.NewTimer(time.Second * 10)
	deleteTimer := time.NewTimer(time.Hour)

	for {
		select {
		case enc, ok = <-logger.tunnel:
			if !ok {
				logger.c <- true
				return
			}

			for _, w := range logger.writers {
				if err := w.Write(enc); err != nil {
					log.Println(err)
				}
			}
			textPool.Put(enc)

		case <-flushTimer.C:
			for _, w := range logger.writers {
				if f, ok := w.(Flusher); ok {
					if err := f.Flush(); err != nil {
						log.Println(err)
					}
				}
			}
			flushTimer.Reset(time.Millisecond * 1000)

		case <-rotateTimer.C:
			for _, w := range logger.writers {
				if r, ok := w.(Rotater); ok {
					if err := r.Rotate(); err != nil {
						log.Println(err)
					}
				}
			}
			rotateTimer.Reset(time.Second* 10)

		case <-deleteTimer.C:
			for _, w := range logger.writers {
				if d, ok := w.(Deleter); ok {
					// delete expired file logic
					if err := d.Delete(); err != nil {
						log.Println(err) // have err but ok
					}
				}
			}
			deleteTimer.Reset(time.Hour)
		}
	}
}

// default
var (
	logger_default *Logger
)

func SetLevel(lvl int) {
	logger_default.level = lvl
}

func SetLayout(layout string) {
	logger_default.layout = layout
}

func Public(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(PUBLIC, fmt, args...)
}

func Trace(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(TRACE, fmt, args...)
}

func Debug(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(DEBUG, fmt, args...)
}

func Warn(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(WARNING, fmt, args...)
}

func Info(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(INFO, fmt, args...)
}

func Error(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(ERROR, fmt, args...)
}

func Fatal(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(FATAL, fmt, args...)
}

func HighTrace(fmt string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(TRACE, fmt, fields...)
}

func HighDebug(fmt string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(DEBUG, fmt, fields...)
}

func HighWarn(fmt string, fields ...interface{}) {
	logger_default.deliverRecordToWriter(WARNING, fmt, fields...)
}

func HighInfo(fmt string, fields ...interface{}) {
	logger_default.deliverRecordToWriter(INFO, fmt, fields...)
}

func HighError(fmt string, fields ...interface{}) {
	logger_default.deliverRecordToWriter(ERROR, fmt, fields...)
}

func HighFatal(fmt string, fields ...interface{}) {
	logger_default.deliverRecordToWriter(FATAL, fmt, fields...)
}

func Register(w Writer) {
	logger_default.Register(w)
}

func (l *Logger) RegisterWithFile(filepath, rotateLogPath string, level int) {
	w := NewFileWriter()
	w.SetFileName(filepath)
	err := w.SetPathPattern(rotateLogPath)
	fmt.Printf("%+v\n", err)
	w.SetLogLevelFloor(level)
	l.Register(w)
}

func NewLoggerWithFile(filepath, rotateLogPath string, level int) *Logger {
	l := NewLogger()
	l.RegisterWithFile(filepath, rotateLogPath, level)
	return l
}

func Close() {
	logger_default.close()
}



//格式化日志
func (l *Logger) formatSliceMsg(keys []string,
	value []interface{}) (str string) {
	splitstr := "||"
	if len(keys) != len(value) {
		return fmt.Sprintf(" keys=%v"+splitstr+"values=%v", keys, value)
	}
	str = splitstr
	for index, v := range keys {
		str += v + "=" + fmt.Sprintf("%+v", value[index]) + splitstr
	}
	str = strings.TrimRight(str, splitstr)
	str = strings.TrimLeft(str, splitstr)
	str = "=> "+str
	return
}

func (l *Logger) TraceSort(keys []string, value []interface{}) {
	if TRACE < l.level {
		return
	}
	msg := l.formatSliceMsg(keys, value)
	l.writeMsg(TRACE, msg)
}

func (l *Logger) DebugSort(keys []string, value []interface{}) {
	if DEBUG < l.level {
		return
	}
	msg := l.formatSliceMsg(keys, value)
	l.writeMsg(DEBUG, msg)
}

func (l *Logger) InfoSort(keys []string, value []interface{}) {
	if INFO < l.level {
		return
	}
	msg := l.formatSliceMsg(keys, value)
	l.writeMsg(INFO, msg)
}

func (l *Logger) WarningSort(keys []string, value []interface{}) {
	if WARNING < l.level {
		return
	}
	msg := l.formatSliceMsg(keys, value)
	l.writeMsg(WARNING, msg)
}

func (l *Logger) ErrorSort(keys []string, value []interface{}) {
	if ERROR < l.level {
		return
	}
	msg := l.formatSliceMsg(keys, value)
	l.writeMsg(ERROR, msg)
}

func (l *Logger) FatalSort(keys []string, value []interface{}) {
	if FATAL < l.level {
		return
	}
	msg := l.formatSliceMsg(keys, value)
	l.writeMsg(FATAL, msg)
}

func (l *Logger) PublicSort(keys []string, value []interface{}) {
	if PUBLIC < l.level {
		return
	}
	msg := l.formatSliceMsg(keys, value)
	l.writeMsg(PUBLIC, msg)
}

//
func (l *Logger) writeMsg(level int, str string) {
	// source code, file and line num
	_, file, line, ok := runtime.Caller(3)
	r := &Record{}
	r.hight = false
	r.info = str
	if ok {
		r.code = path.Base(file)
		r.line = line
	}
	r.time = time.Now()
	r.level = level
	enc := textPool.Get().(*textEncoder)
	enc.truncate()
	enc.level = r.level
	r.Bytes(enc)

	l.tunnel <- enc
	return
}

func ListTrace(keys []string, value []interface{}) {
	logger_default.TraceSort(keys, value)
}

func ListDebug(keys []string, value []interface{}) {
	logger_default.DebugSort(keys, value)
}

func ListInfo(keys []string, value []interface{}) {
	logger_default.InfoSort(keys, value)
}

func ListWarning(keys []string, value []interface{}) {
	logger_default.WarningSort(keys, value)
}

func ListError(keys []string, value []interface{}) {
	logger_default.ErrorSort(keys, value)
}

func ListFatal(keys []string, value []interface{}) {
	logger_default.FatalSort(keys, value)
}

func ListPublic(keys []string, value []interface{}) {
	logger_default.PublicSort(keys, value)
}

func TraceTrace(trace_id string, keys []string, value []interface{}) {
	keys = append(keys, "trace_id")
	value = append(value, trace_id)
	logger_default.TraceSort(keys, value)
}

func TraceDebug(trace_id string, keys []string, value []interface{}) {
	keys = append(keys, "trace_id")
	value = append(value, trace_id)
	logger_default.DebugSort(keys, value)
}

func TraceInfo(trace_id string, keys []string, value []interface{}) {
	keys = append(keys, "trace_id")
	value = append(value, trace_id)
	logger_default.InfoSort(keys, value)
}

func TraceError(trace_id string, keys []string, value []interface{}) {
	keys = append(keys, "trace_id")
	value = append(value, trace_id)
	logger_default.ErrorSort(keys, value)
}

func TraceWarning(trace_id string, keys []string, value []interface{}) {
	keys = append(keys, "trace_id")
	value = append(value, trace_id)
	logger_default.WarningSort(keys, value)
}

func TraceFatal(trace_id string, keys []string, value []interface{}) {
	keys = append(keys, "trace_id")
	value = append(value, trace_id)
	logger_default.FatalSort(keys, value)
}

func TracePublic(trace_id string, keys []string, value []interface{}) {
	keys = append(keys, "trace_id")
	value = append(value, trace_id)
	logger_default.PublicSort(keys, value)
}

//LoggerContext package
type LoggerContext struct {
	lg     *Logger
	logID  string
	detail map[string]interface{}
}

//NewLoggerContext 初始化一个上线访问日志
func NewLoggerContext(logid string) *LoggerContext {
	return &LoggerContext{
		lg:     logger_default,
		logID:  logid,
		detail: make(map[string]interface{}),
	}
}

//fmt log
func (lc *LoggerContext) msgFormat(level int, keys []string,
	values []interface{}) (str string) {
	for key, val := range lc.detail {
		keys = append(keys, key)
		values = append(values, val)
	}
	keys = append(keys, "logID")
	values = append(values, lc.logID)
	splitstr := "||"
	if len(keys) != len(values) {
		return fmt.Sprintf(" keys=%v"+splitstr+" values=%v", keys, values)
	}
	str = splitstr
	for index, v := range keys {
		str += v + "= " + fmt.Sprintf("%+v", values[index]) + splitstr
	}
	str = strings.TrimRight(str, splitstr)
	str = strings.TrimLeft(str,splitstr)
	str = ">>"+str
	lc.lg.writeMsg(level, str)
	return
}

//
func (lc *LoggerContext) LogInfo(keys []string, value []interface{}) {
	if INFO < lc.lg.level { // level 是固定的，不能改变
		return
	}
	lc.msgFormat(INFO, keys, value)
}

func (lc *LoggerContext) LogError(keys []string, value []interface{}) {
	if ERROR < lc.lg.level {
		return
	}
	lc.msgFormat(ERROR, keys, value)
}

func (lc *LoggerContext) LogDebug(keys []string, value []interface{}) {
	if DEBUG < lc.lg.level {
		return
	}
	lc.msgFormat(DEBUG, keys, value)
}

//RandNumString generate random logId
func RandNumString(lenNum int) string {
	str := "1234567890"
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	ret := make([]byte, lenNum)
	length := len(str)
	for i := 0; i < lenNum; i++ {
		pos := rnd.Intn(length)
		ret[i] = str[pos]
	}
	return string(ret)
}
