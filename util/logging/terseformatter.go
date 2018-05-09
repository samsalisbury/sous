package logging

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	terseFormatter struct {
		StackTraceField string
		priorityFields  []string
		omittedFields   map[string]struct{}
	}
)

const defaultTimestampFormat = time.StampMilli

func newTerseFormatter(prio, omit []FieldName) *terseFormatter {
	pr := []string{}
	om := map[string]struct{}{}

	for _, p := range prio {
		pr = append(pr, string(p))
		om[string(p)] = struct{}{}
	}

	for _, o := range omit {
		om[string(o)] = struct{}{}
	}

	return &terseFormatter{
		priorityFields: pr,
		omittedFields:  om,
	}
}

var traceShortener = regexp.MustCompile(`(?m)^(github.com/opentable/sous/)?`)

func (f *terseFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestampFormat := defaultTimestampFormat
	b.WriteString(entry.Time.Format(timestampFormat))
	b.WriteString(": [")
	b.WriteString(strings.ToUpper(entry.Level.String())[0:3])
	b.WriteString("] ")
	if entry.Message != "" {
		b.WriteString(entry.Message)
	}
	for _, prio := range f.priorityFields {
		if val, has := entry.Data[prio]; has {
			f.appendKeyValue(b, prio, val)
		}
	}
	for _, key := range keys {
		if _, omit := f.omittedFields[key]; !omit {
			f.appendKeyValue(b, key, entry.Data[key])
		}
	}

	b.WriteByte('\n')
	if stack, hasSTF := entry.Data[f.StackTraceField]; f.StackTraceField != "" && hasSTF {
		ss := stack.(string)
		ss = traceShortener.ReplaceAllString(ss, "\t")
		b.WriteString(ss)
		b.WriteByte('\n')
	}

	return b.Bytes(), nil
}

func (f *terseFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
}

func (f *terseFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}

func (f *terseFormatter) needsQuoting(text string) bool {
	if len(text) == 0 {
		return true
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}
