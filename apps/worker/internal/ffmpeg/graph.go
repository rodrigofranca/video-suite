package ffmpeg

import (
	"fmt"
	"strconv"
	"strings"
)

// Graph builds an FFmpeg filtergraph and the inputs it reads (docs/adr/0035).
// Filters are applied to streams and give new streams back, so whoever adds
// a filter needs no labels and no knowledge of what came before. Every
// stream can be read once, as FFmpeg requires; reading one twice is an
// error that Args reports.
type Graph struct {
	inputs [][]string // per input: its options, "-i" and the path
	chains []chain
	labels int
	read   map[string]bool
	err    error
}

// Stream is a stream in the graph: an input's ("0:v:0") or a filter's
// output (a label such as "s3").
type Stream struct {
	name  string
	input bool
}

// Filter is one filter and its options, as FFmpeg writes it: scale=1080:1920.
type Filter struct {
	Name string
	Opts []string
}

// F makes a filter. Options can be strings ("w=1080" or "1080"), ints or
// floats (written with three decimals).
func F(name string, opts ...any) Filter {
	f := Filter{Name: name}
	for _, o := range opts {
		switch v := o.(type) {
		case string:
			f.Opts = append(f.Opts, v)
		case int:
			f.Opts = append(f.Opts, strconv.Itoa(v))
		case float64:
			f.Opts = append(f.Opts, strconv.FormatFloat(v, 'f', 3, 64))
		default:
			f.Opts = append(f.Opts, fmt.Sprint(v))
		}
	}
	return f
}

func (f Filter) String() string {
	if len(f.Opts) == 0 {
		return f.Name
	}
	return f.Name + "=" + strings.Join(f.Opts, ":")
}

type chain struct {
	in      []Stream
	filters []Filter
	out     []Stream
}

// Input adds an input file, with options that apply to it alone (such as
// -ss and -t), and returns its index.
func (g *Graph) Input(path string, opts ...string) int {
	g.inputs = append(g.inputs, append(append([]string{}, opts...), "-i", path))
	return len(g.inputs) - 1
}

// Video is an input's first video stream; Audio its first audio stream.
func (g *Graph) Video(input int) Stream {
	return Stream{name: fmt.Sprintf("%d:v:0", input), input: true}
}
func (g *Graph) Audio(input int) Stream {
	return Stream{name: fmt.Sprintf("%d:a:0", input), input: true}
}

// Apply runs filters one after another over the streams in and returns
// the stream they make. With no streams in, the first filter is a source
// (such as anullsrc). A single stream that the last filters made is
// continued in their chain, which keeps the graph short.
func (g *Graph) Apply(in []Stream, filters ...Filter) Stream { return g.Fork(in, 1, filters...)[0] }

// Pipe is Apply over one stream.
func (g *Graph) Pipe(in Stream, filters ...Filter) Stream { return g.Apply([]Stream{in}, filters...) }

// Fork runs filters that make several streams, such as split or concat,
// and returns them in order.
func (g *Graph) Fork(in []Stream, outputs int, filters ...Filter) []Stream {
	g.readAll(in)
	out := make([]Stream, outputs)
	for i := range out {
		out[i] = g.label()
	}
	if n := len(g.chains); len(in) == 1 && n > 0 && len(g.chains[n-1].out) == 1 && g.chains[n-1].out[0] == in[0] {
		last := &g.chains[n-1]
		last.filters = append(last.filters, filters...)
		last.out = out
		return out
	}
	g.chains = append(g.chains, chain{in: in, filters: filters, out: out})
	return out
}

// Split makes n copies of a stream.
func (g *Graph) Split(in Stream, n int) []Stream { return g.Fork([]Stream{in}, n, F("split", n)) }

func (g *Graph) label() Stream {
	s := Stream{name: "s" + strconv.Itoa(g.labels)}
	g.labels++
	return s
}

func (g *Graph) readAll(in []Stream) {
	if g.read == nil {
		g.read = map[string]bool{}
	}
	for _, s := range in {
		if g.read[s.name] && g.err == nil {
			g.err = fmt.Errorf("filter graph: stream [%s] is read twice", s.name)
		}
		g.read[s.name] = true
	}
}

// String is the graph as -filter_complex takes it.
func (g *Graph) String() string {
	parts := make([]string, len(g.chains))
	for i, c := range g.chains {
		var b strings.Builder
		for _, s := range c.in {
			b.WriteString("[" + s.name + "]")
		}
		for j, f := range c.filters {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString(f.String())
		}
		for _, s := range c.out {
			b.WriteString("[" + s.name + "]")
		}
		parts[i] = b.String()
	}
	return strings.Join(parts, ";")
}

// Args returns the inputs, the graph (when there are filters) and a -map for
// each stream to keep, in order. An input's audio is mapped as optional,
// since a source may have no sound.
func (g *Graph) Args(keep ...Stream) ([]string, error) {
	if g.err != nil {
		return nil, g.err
	}
	var args []string
	for _, in := range g.inputs {
		args = append(args, in...)
	}
	if len(g.chains) > 0 {
		args = append(args, "-filter_complex", g.String())
	}
	for _, s := range keep {
		switch {
		case !s.input:
			args = append(args, "-map", "["+s.name+"]")
		case strings.Contains(s.name, ":a:"):
			args = append(args, "-map", s.name+"?")
		default:
			args = append(args, "-map", s.name)
		}
	}
	return args, nil
}

// Escape writes a value for a filter option, escaping the characters that
// separate options, filters and labels.
func Escape(s string) string { return filterEscaper.Replace(s) }

var filterEscaper = strings.NewReplacer(`\`, `\\\\`, `'`, `\\\'`, `:`, `\\:`, `,`, `\,`, `;`, `\;`, `[`, `\[`, `]`, `\]`)
