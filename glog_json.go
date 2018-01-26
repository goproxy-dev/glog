// Go support for leveled logs, analogous to https://code.google.com/p/google-glog/
//
// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Json for logs.

package glog

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

var JVersion string
var JFieldPrefix string
var JEscapeHTML bool

type Json struct {
	buf  [1024]byte
	p    []byte
	s    severity
	file string
	line int
}

func J(level Level) *Json {
	return json(infoLog, level)
}

func JInfo() *Json {
	return json(infoLog, 0)
}

func JWarning() *Json {
	return json(warningLog, 0)
}

func JError() *Json {
	return json(errorLog, 0)
}

func JFatal() *Json {
	return json(fatalLog, 0)
}

func (j *Json) Int(name string, i int) *Json {
	return j.Int64(name, int64(i))
}

func (j *Json) Int8(name string, i int8) *Json {
	return j.Int64(name, int64(i))
}

func (j *Json) Int16(name string, i int16) *Json {
	return j.Int64(name, int64(i))
}

func (j *Json) Int32(name string, i int32) *Json {
	return j.Int64(name, int64(i))
}

func (j *Json) Uint(name string, i uint) *Json {
	return j.Uint64(name, uint64(i))
}

func (j *Json) Uint8(name string, i uint8) *Json {
	return j.Uint64(name, uint64(i))
}

func (j *Json) Uint16(name string, i uint16) *Json {
	return j.Uint64(name, uint64(i))
}

func (j *Json) Uint32(name string, i uint32) *Json {
	return j.Uint64(name, uint64(i))
}

func (j *Json) Int64(name string, i int64) *Json {
	if j != nil {
		j.p = append(j.p, ',', '"')
		if JFieldPrefix != "" {
			j.p = append(j.p, JFieldPrefix...)
		}
		j.p = append(j.p, name...)
		j.p = append(j.p, '"', ':')
		j.p = strconv.AppendInt(j.p, i, 10)
	}
	return j
}

func (j *Json) Uint64(name string, i uint64) *Json {
	if j != nil {
		j.p = append(j.p, ',', '"')
		if JFieldPrefix != "" {
			j.p = append(j.p, JFieldPrefix...)
		}
		j.p = append(j.p, name...)
		j.p = append(j.p, '"', ':')
		j.p = strconv.AppendUint(j.p, i, 10)
	}
	return j
}

func (j *Json) Float32(name string, f float64) *Json {
	if j != nil {
		j.p = append(j.p, ',', '"')
		if JFieldPrefix != "" {
			j.p = append(j.p, JFieldPrefix...)
		}
		j.p = append(j.p, name...)
		j.p = append(j.p, '"', ':')
		j.p = strconv.AppendFloat(j.p, f, 'f', -1, 32)
	}
	return j
}

func (j *Json) Float64(name string, f float64) *Json {
	if j != nil {
		j.p = append(j.p, ',', '"')
		if JFieldPrefix != "" {
			j.p = append(j.p, JFieldPrefix...)
		}
		j.p = append(j.p, name...)
		j.p = append(j.p, '"', ':')
		j.p = strconv.AppendFloat(j.p, f, 'f', -1, 64)
	}
	return j
}

func (j *Json) Bool(name string, b bool) *Json {
	if j != nil {
		j.p = append(j.p, ',', '"')
		if JFieldPrefix != "" {
			j.p = append(j.p, JFieldPrefix...)
		}
		j.p = append(j.p, name...)
		if b {
			j.p = append(j.p, '"', ':', 't', 'r', 'u', 'e')
		} else {
			j.p = append(j.p, '"', ':', 'f', 'a', 'l', 's', 'e')
		}
	}
	return j
}

func (j *Json) Err(err error) *Json {
	if j != nil {
		j.p = append(j.p, ",\"error\":"...)
		j.string(err.Error(), JEscapeHTML)
	}
	return j
}

func (j *Json) Str(name string, s string) *Json {
	if j != nil {
		j.p = append(j.p, ',', '"')
		if JFieldPrefix != "" {
			j.p = append(j.p, JFieldPrefix...)
		}
		j.p = append(j.p, name...)
		j.p = append(j.p, '"', ':')
		j.string(s, JEscapeHTML)
	}
	return j
}

func (j *Json) Strs(name string, ss []string) *Json {
	if j != nil {
		j.p = append(j.p, '"')
		if JFieldPrefix != "" {
			j.p = append(j.p, JFieldPrefix...)
		}
		j.p = append(j.p, name...)
		j.p = append(j.p, '"', ':', '[')
		for i, s := range ss {
			j.string(s, JEscapeHTML)
			if i < len(ss)-1 {
				j.p = append(j.p, ',')
			}
		}
		j.p = append(j.p, ']', ',')
	}
	return j
}

func (j *Json) Msg(s string) {
	if j != nil {
		if s != "" {
			j.p = append(j.p, ",\"message\":"...)
			j.string(s, false)
		}
		j.p = append(j.p, '}', '\n')
		logging.outputj(j.s, j, j.file, j.line, false)
	}
}

func (j *Json) Msgf(format string, v ...interface{}) {
	if j == nil {
		j.Msg(fmt.Sprintf(format, v...))
	}
}

func json(s severity, level Level) *Json {
	_, file, line, ok := runtime.Caller(2)

	if !ok {
		file = "???"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}
	}

	if line < 0 {
		line = 0 // not a real line number, but acceptable to someDigits
	}

	if logging.verbosity.get() >= level {
		return jheader(s, level, file, line)
	}

	if atomic.LoadInt32(&logging.filterLength) > 0 {
		// Now we need a proper lock to use the logging structure. The pcs field
		// is shared so we must lock before accessing it. This is fairly expensive,
		// but if V logging is enabled we're slow anyway.
		logging.mu.Lock()
		defer logging.mu.Unlock()
		if runtime.Callers(2, logging.pcs[:]) == 0 {
			return nil
		}
		v, ok := logging.vmap[logging.pcs[0]]
		if !ok {
			v = logging.setV(logging.pcs[0])
		}

		if v >= level {
			return jheader(s, level, file, line)
		}
	}

	return nil
}

var jbufpool = sync.Pool{
	New: func() interface{} {
		return new(Json)
	},
}

func jheader(s severity, level Level, file string, line int) *Json {
	now := timeNow()
	j := jbufpool.Get().(*Json)

	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	_, offset := now.Zone()

	j.buf[0] = '{'
	j.buf[1] = '"'
	j.buf[2] = 't'
	j.buf[3] = 'i'
	j.buf[4] = 'm'
	j.buf[5] = 'e'
	j.buf[6] = '"'
	j.buf[7] = ':'
	j.buf[8] = '"'
	j.buf[12] = digits[year%10]
	year /= 10
	j.buf[11] = digits[year%10]
	year /= 10
	j.buf[10] = digits[year%10]
	year /= 10
	j.buf[9] = digits[year%10]
	j.buf[13] = '-'
	j.twoDigits(14, int(month))
	j.buf[16] = '-'
	j.twoDigits(17, day)
	j.buf[19] = 'T'
	j.twoDigits(20, hour)
	j.buf[22] = ':'
	j.twoDigits(23, minute)
	j.buf[25] = ':'
	j.twoDigits(26, second)
	if offset > 0 {
		j.buf[28] = '+'
		offset /= 60
	} else {
		j.buf[28] = '-'
		offset = (-offset) / 60
	}
	j.twoDigits(29, offset/60)
	j.buf[31] = ':'
	j.twoDigits(32, offset%60)
	j.buf[34] = '"'
	j.buf[35] = ','
	j.buf[36] = '"'
	j.buf[37] = 't'
	j.buf[38] = 'i'
	j.buf[39] = 'm'
	j.buf[40] = 'e'
	j.buf[41] = 's'
	j.buf[42] = 't'
	j.buf[43] = 'a'
	j.buf[44] = 'm'
	j.buf[45] = 'p'
	j.buf[46] = '"'
	j.buf[47] = ':'
	j.timestamp(48, now)
	j.p = j.buf[:65]
	j.p = append(j.p, ",\"level\":\""...)
	j.p = append(j.p, lowerSeverityName[s]...)
	if JVersion != "" {
		j.p = append(j.p, "\",\"version\":\""...)
		j.p = append(j.p, JVersion...)
	}
	j.p = append(j.p, "\",\"host\":\""...)
	j.p = append(j.p, host...)
	j.p = append(j.p, "\",\"pid\":"...)
	j.p = strconv.AppendInt(j.p, int64(pid), 10)
	j.p = append(j.p, ",\"file\":\""...)
	j.p = append(j.p, file...)
	j.p = append(j.p, "\",\"line\":"...)
	j.p = strconv.AppendInt(j.p, int64(line), 10)

	return j
}

func (l *loggingT) outputj(s severity, buf *Json, file string, line int, alsoToStderr bool) {
	l.mu.Lock()
	if l.traceLocation.isSet() {
		if l.traceLocation.match(file, line) {
			buf.p = append(buf.p, stacks(false)...)
		}
	}
	data := buf.p
	if !flag.Parsed() {
		os.Stderr.Write([]byte("ERROR: logging before flag.Parse: "))
		os.Stderr.Write(data)
	} else if l.toStderr {
		os.Stderr.Write(data)
	} else {
		if alsoToStderr || l.alsoToStderr || s >= l.stderrThreshold.get() {
			os.Stderr.Write(data)
		}
		if l.file[s] == nil {
			if err := l.createFiles(s); err != nil {
				os.Stderr.Write(data) // Make sure the message appears somewhere.
				l.exit(err)
			}
		}
		switch s {
		case fatalLog:
			l.file[fatalLog].Write(data)
			fallthrough
		case errorLog:
			l.file[errorLog].Write(data)
			fallthrough
		case warningLog:
			l.file[warningLog].Write(data)
			fallthrough
		case infoLog:
			l.file[infoLog].Write(data)
		}
	}
	if s == fatalLog {
		// If we got here via Exit rather than Fatal, print no stacks.
		if atomic.LoadUint32(&fatalNoStacks) > 0 {
			l.mu.Unlock()
			timeoutFlush(10 * time.Second)
			os.Exit(1)
		}
		// Dump all goroutine stacks before exiting.
		// First, make sure we see the trace for the current goroutine on standard error.
		// If -logtostderr has been specified, the loop below will do that anyway
		// as the first stack in the full dump.
		if !l.toStderr {
			os.Stderr.Write(stacks(false))
		}
		// Write the stack trace for all goroutines to the files.
		trace := stacks(true)
		logExitFunc = func(error) {} // If we get a write error, we'll still exit below.
		for log := fatalLog; log >= infoLog; log-- {
			if f := l.file[log]; f != nil { // Can be nil if -logtostderr is set.
				f.Write(trace)
			}
		}
		l.mu.Unlock()
		timeoutFlush(10 * time.Second)
		os.Exit(255) // C++ uses -1, which is silly because it's anded with 255 anyway.
	}
	jbufpool.Put(buf)
	l.mu.Unlock()
	if stats := severityStats[s]; stats != nil {
		atomic.AddInt64(&stats.lines, 1)
		atomic.AddInt64(&stats.bytes, int64(len(data)))
	}
}

func (j *Json) twoDigits(i, d int) {
	j.buf[i+1] = digits[d%10]
	d /= 10
	j.buf[i] = digits[d%10]
}

func (j *Json) timestamp(i int, now time.Time) int {
	i += 17

	d := int64(now.Nanosecond() / 1000)
	k := 6
	for ; d != 0; k-- {
		i--
		j.buf[i] = digits[d%10]
		d /= 10
	}
	for ; k != 0; k-- {
		i--
		j.buf[i] = '0'
	}

	i--
	j.buf[i] = '.'

	d = now.Unix()
	for d != 0 {
		i--
		j.buf[i] = digits[d%10]
		d /= 10
	}

	return i
}

var hex = "0123456789abcdef"

// https://golang.org/src/encoding/json/encode.go
func (j *Json) string(s string, escapeHTML bool) {
	j.p = append(j.p, '"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if htmlSafeSet[b] || (!escapeHTML && safeSet[b]) {
				i++
				continue
			}
			if start < i {
				j.p = append(j.p, s[start:i]...)
			}
			switch b {
			case '\\', '"':
				j.p = append(j.p, '\\', b)
			case '\n':
				j.p = append(j.p, '\\', 'n')
			case '\r':
				j.p = append(j.p, '\\', 'r')
			case '\t':
				j.p = append(j.p, '\\', 't')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				j.p = append(j.p, '\\', 'u', '0', '0', hex[b>>4], hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				j.p = append(j.p, s[start:i]...)
			}
			j.p = append(j.p, '\\', 'u', 'f', 'f', 'f', 'd')
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				j.p = append(j.p, s[start:i]...)
			}
			j.p = append(j.p, '\\', 'u', '2', '0', '2', hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		j.p = append(j.p, s[start:]...)
	}
	j.p = append(j.p, '"')
}

var lowerSeverityName = []string{
	infoLog:    "info",
	warningLog: "warning",
	errorLog:   "error",
	fatalLog:   "fatal",
}

// safeSet holds the value true if the ASCII character with the given array
// position can be represented inside a JSON string without any further
// escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}

// htmlSafeSet holds the value true if the ASCII character with the given
// array position can be safely represented inside a JSON string, embedded
// inside of HTML <script> tags, without any additional escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), the backslash character ("\"), HTML opening and closing
// tags ("<" and ">"), and the ampersand ("&").
var htmlSafeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      false,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      false,
	'=':      true,
	'>':      false,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}
