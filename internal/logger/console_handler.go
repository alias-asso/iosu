// Package logger contains the structures used by gokord's custom slog.Logger.
//
// If you want to modify the line logged in the output, see NewContext.
//
// COPYRIGHT BSD-3-Clause NYTTIKORD et al.: https://github.com/nyttikord/gokord
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	AnsiReset       = "\033[0m"
	AnsiRed         = "\033[38;5;9m"
	AnsiGrey        = "\033[38;5;244m"
	AnsiGreen       = "\033[38;5;2m"
	AnsiYellow      = "\033[38;5;11m"
	AnsiBlue        = "\033[38;5;6m"
	AnsiMagenta     = "\033[38;5;13m"
	AnsiCyan        = "\033[38;5;14m"
	AnsiWhite       = "\033[37m"
	AnsiBlueBold    = "\033[34;1m"
	AnsiMagentaBold = "\033[35;1m"
	AnsiRedBold     = "\033[31;1m"
	AnsiYellowBold  = "\033[33;1m"

	AnsiNotImportant = AnsiGrey
)

// ConsoleHandler represents the default slog.Handler used by gokord.
//
// See New to create a new ConsoleHandler with the given Options.
type ConsoleHandler struct {
	opts              Options
	goas              []groupOrAttrs
	mu                *sync.Mutex
	out               io.Writer
	maxFileLineLength *int
}

// Options of the ConsoleHandler.
type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler

	// MaxFileLineLength is the maximum length of the caller part.
	// Default value is 25.
	MaxFileLineLength int
	// If Align, everything logged will be aligned dynamically.
	Align bool
	// If ArgsAreImportant, args are in default terminal color.
	// If not, they are in AnsiNotImportant (default).
	ArgsAreImportant bool
	// If TrimVersion, package versions are removed from the caller part.
	TrimVersion bool
}

// New creates a new ConsoleHandler.
func New(out io.Writer, opts *Options) *ConsoleHandler {
	h := &ConsoleHandler{out: out, mu: new(sync.Mutex), maxFileLineLength: new(int)}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	if h.opts.MaxFileLineLength == 0 {
		h.opts.MaxFileLineLength = 30
	}
	return h
}

type key int

const (
	callerSkipKey key = 0
)

var maxLength = max(
	len(slog.LevelDebug.String()),
	len(slog.LevelInfo.String()),
	len(slog.LevelWarn.String()),
	len(slog.LevelError.String()),
)

// NewContext returns a new context.Context with the callerSkip given.
//
// callerSkip is the number of runtime calls to log before this one.
// 0 is for the current.
// 1 is for the precedent call.
// n is for the n times precedent call.
// The calls to the log is already skipped.
//
// See FromContext to extract the caller from a context.Context.
func NewContext(ctx context.Context, callerSkip int) context.Context {
	return context.WithValue(ctx, callerSkipKey, callerSkip)
}

// FromContext returns the caller in the given context.Context.
//
// See NewContext to create a context.Context.
func FromContext(ctx context.Context) (int, bool) {
	caller, ok := ctx.Value(callerSkipKey).(int)
	return caller, ok
}

func color(level slog.Level) string {
	if level >= slog.LevelError {
		return AnsiRed
	} else if level >= slog.LevelWarn {
		return AnsiYellow
	} else if level >= slog.LevelInfo {
		return AnsiGreen
	} else {
		return AnsiReset
	}
}

// Enabled indicates if the given slog.Level is enabled.
func (h *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle a slog.Record.
func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	buf := make([]byte, 0, 1024)
	if !r.Time.IsZero() {
		buf = fmt.Appendf(buf, "%s%s%s ", AnsiNotImportant, r.Time.Format(time.DateTime), AnsiReset)
	}
	sp := " "
	if h.opts.Align {
		for range maxLength - len(r.Level.String()) {
			sp += " "
		}
	}
	buf = fmt.Appendf(buf, "[%s%s%s]%s", color(r.Level), r.Level, AnsiReset, sp)
	if r.PC != 0 {
		caller, ok := FromContext(ctx)
		var file string
		var line int
		if ok {
			_, file, line, ok = runtime.Caller(caller + 3)
		} else {
			_, file, line, ok = runtime.Caller(3)
		}
		files := strings.Split(file, "/")
		if len(files) == 1 {
			file = files[len(files)-1]
		} else {
			// remove package version from log
			packge := files[len(files)-2]
			i := strings.Index(packge, "@")
			if !h.opts.TrimVersion || i == -1 {
				i = len(packge)
			}
			file = packge[:i] + "/" + files[len(files)-1]
		}

		fileLine := fmt.Sprintf("%s:%d", file, line)
		sp = " "
		if h.opts.Align {
			if len(fileLine) > h.opts.MaxFileLineLength {
				lineStr := strconv.Itoa(line)
				fileLine = fmt.Sprintf("...%s:%s", file[4+len(lineStr)+len(file)-h.opts.MaxFileLineLength:], lineStr)
			}
			*h.maxFileLineLength = max(len(fileLine), *h.maxFileLineLength)
			for range *h.maxFileLineLength - len(fileLine) {
				sp += " "
			}
		}
		buf = fmt.Appendf(buf, "%s%s%s- %s", AnsiNotImportant, fileLine, sp, AnsiReset)
	}
	buf = fmt.Appendf(buf, "%s%s%s", color(r.Level), r.Message, AnsiReset)
	if !h.opts.ArgsAreImportant {
		buf = fmt.Appendf(buf, "%s", AnsiNotImportant)
	}
	// Handle state from WithGroup and WithAttrs.
	goas := h.goas
	if r.NumAttrs() == 0 {
		// If the record has no Attrs, remove groups at the end of the list;
		// they are empty.
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}
	for _, goa := range goas {
		if goa.group != "" {
			buf = fmt.Appendf(buf, " %s={", goa.group)
		} else {
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, a)
			}
			//buf = fmt.Appendf(buf, "}") // I don't know where I should put it
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})
	buf = fmt.Appendf(buf, "%s\n", AnsiReset)
	_, err := h.out.Write(buf)
	return err
}

func (h *ConsoleHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	buf = fmt.Appendf(buf, " ")
	a.Key = escapeSpace(a.Key)
	buf = fmt.Appendf(buf, "%s=", a.Key)
	if val, ok := a.Value.Any().(fmt.Stringer); ok {
		return fmt.Appendf(buf, "%s", escapeSpace(val.String()))
	}
	if val, ok := a.Value.Any().([]byte); ok {
		return fmt.Appendf(buf, "%s", escapeSpace(string(val)))
	}
	if val, ok := a.Value.Any().(json.RawMessage); ok {
		return fmt.Appendf(buf, "%s", escapeSpace(string(val)))
	}
	if val, ok := a.Value.Any().(error); ok {
		return fmt.Appendf(buf, "%s", escapeSpace(val.Error()))
	}
	switch a.Value.Kind() {
	case slog.KindString:
		buf = fmt.Appendf(buf, "%s", escapeSpace(a.Value.String()))
	case slog.KindTime:
		buf = fmt.Appendf(buf, "%s", a.Value.Time().Format(time.RFC3339))
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		if a.Key != "" {
			buf = fmt.Appendf(buf, "%s={", a.Key)
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
		if a.Key != "" {
			buf[len(buf)-1] = '}' // replace last space by }
		}
	default:
		buf = fmt.Appendf(buf, "%v", a.Value.Any())
	}
	return buf
}

func escapeSpace(s string) string {
	if strings.Contains(s, " ") {
		s = fmt.Sprintf("%q", s)
	}
	return s
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func (h *ConsoleHandler) withGroupOrAttrs(goa groupOrAttrs) *ConsoleHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}
