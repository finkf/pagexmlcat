package main // import "github.com/finkf/pagexmlcat"

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
)

var (
	words   bool
	regions bool
	id      bool
	conf    bool
	serial  bool
	pfn     bool
	norm    bool
	indices index
)

func init() {
	flag.BoolVar(&words, "words", false, "cat words")
	flag.BoolVar(&regions, "regions", false, "cat regions")
	flag.BoolVar(&id, "id", false, "print id header")
	flag.BoolVar(&conf, "conf", false, "print confidence")
	flag.BoolVar(&serial, "serial", false, "ignore region ordering")
	flag.BoolVar(&pfn, "filename", false, "print filenames")
	flag.BoolVar(&norm, "norm", false, "normalize output")
	flag.Var(&indices, "index", "set indices")
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "%s [OPTIONS] [FILES...]\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	chk(flag.Set("index", "0"))
	flag.Parse()
	for _, arg := range flag.Args() {
		chk(printFile(arg))
	}
	if len(flag.Args()) == 0 {
		chk(print(os.Stdout, os.Stdin, ""))
	}
}

func printFile(path string) error {
	if path == "-" {
		return print(os.Stdout, os.Stdin, "")
	}
	in, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot print file: %v", err)
	}
	defer in.Close()
	if err := print(os.Stdout, in, path); err != nil {
		return fmt.Errorf("cannot print file %s: %v", path, err)
	}
	return nil
}

func print(out io.Writer, in io.Reader, name string) error {
	doc, err := xmlquery.Parse(in)
	if err != nil {
		return fmt.Errorf("cannot print: %v", err)
	}
	if serial { // ignore region ordering
		return printSegs(out, doc, name)
	}
	xpath := "//*[local-name()='OrderedGroup']/*[local-name()='RegionRefIndexed']"
	rris := xmlquery.Find(doc, xpath)
	if len(rris) == 0 { // no region ordering defined
		return printSegs(out, doc, name)
	}
	regionRefs := make([]regionRef, len(rris))
	for i, node := range rris {
		rr, err := newRegionRef(node)
		if err != nil {
			return fmt.Errorf("invalid RegionRefIndexed: %v", err)
		}
		regionRefs[i] = rr
	}
	return printOrdered(out, doc, regionRefs, name)
}

func printOrdered(out io.Writer, node *xmlquery.Node, refs []regionRef, name string) error {
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].index < refs[j].index
	})
	xpathfmt := "//*[local-name()='TextRegion'][@id=%q]"
	for _, ref := range refs {
		nodes := xmlquery.Find(node, fmt.Sprintf(xpathfmt, ref.ref))
		for _, node := range nodes {
			if err := printSegs(out, node, name); err != nil {
				return fmt.Errorf("cannot print ordered: %v", err)
			}
		}
	}
	return nil
}

func printSegs(out io.Writer, node *xmlquery.Node, name string) error {
	seg := "TextLine"
	if words {
		seg = "Word"
	} else if regions {
		seg = "TextRegion"
	}
	segs := xmlquery.Find(node, fmt.Sprintf("//*[local-name()=%q]", seg))
	for _, node := range segs {
		if err := printTextEquivs(out, node, name); err != nil {
			return fmt.Errorf("cannot print: %v", err)
		}
	}
	return nil
}

func printTextEquivs(out io.Writer, node *xmlquery.Node, name string) error {
	tes := xmlquery.Find(node, fmt.Sprintf("/*[local-name()='TextEquiv']"))
	for _, index := range indices {
		if index >= len(tes) || -index >= len(tes) {
			return fmt.Errorf("cannot print text equiv: invalid index %d", index)
		}
		if index < 0 {
			index = len(tes) + index // index < 0
		}
		if pfn && name != "" {
			if _, err := fmt.Fprintf(out, "%s ", name); err != nil {
				return fmt.Errorf("cannot print text equiv: cannot print filename: %v", err)
			}
		}
		for i := 0; id && i < len(node.Attr); i++ {
			if node.Attr[i].Name.Local != "id" {
				continue
			}
			if _, err := fmt.Fprintf(out, "%s@%d ", node.Attr[i].Value, index); err != nil {
				return fmt.Errorf("cannot print text equiv: cannot print id: %v", err)
			}
			break
		}
		if err := printUnicode(out, tes[index]); err != nil {
			return fmt.Errorf("cannot print text equiv: %v", err)
		}
	}
	return nil
}

func printUnicode(out io.Writer, node *xmlquery.Node) error {
	for i := 0; conf && i < len(node.Attr); i++ {
		if node.Attr[i].Name.Local != "conf" {
			continue
		}
		if _, err := fmt.Fprintf(out, "%s ", node.Attr[i].Value); err != nil {
			return fmt.Errorf("cannot print text equiv: cannot print conf: %v", err)
		}
		break
	}
	uni := xmlquery.Find(node, "/*[local-name()='Unicode']")
	if len(uni) == 0 {
		return fmt.Errorf("cannot print unicode: missing")
	}
	// If first child is nil, the unicode node holds empty text.
	var text string
	if uni[0].FirstChild != nil {
		text = uni[0].FirstChild.Data
	}
	if norm {
		text = strings.ReplaceAll(text, " ", "_")
	}
	if _, err := fmt.Fprintln(out, text); err != nil {
		return fmt.Errorf("cannot print unicode: %v", err)
	}
	return nil
}

// regionRef is a struct
type regionRef struct {
	ref   string // ref (id) of the region reference
	index int    // order of the regeion reference
}

func newRegionRef(node *xmlquery.Node) (regionRef, error) {
	var ret regionRef
	var refFound, indexFound bool
	for _, attr := range node.Attr {
		switch attr.Name.Local {
		case "regionRef":
			ret.ref = attr.Value
			refFound = true
		case "index":
			index, err := strconv.Atoi(attr.Value)
			if err != nil {
				return ret, fmt.Errorf("invalid index %s: %v", attr.Value, err)
			}
			ret.index = index
			indexFound = true
		}
	}
	if !refFound {
		return ret, fmt.Errorf("missing regionRef attribute")
	}
	if !indexFound {
		return ret, fmt.Errorf("missing index attribute")
	}
	return ret, nil
}

type index []int

func (i *index) String() string {
	strs := make([]string, len(*i))
	for k := range *i {
		strs[k] = strconv.Itoa((*i)[k])
	}
	return strings.Join(strs, ",")
}

func (i *index) Set(val string) error {
	strs := strings.Split(val, ",")
	*i = make([]int, len(strs))
	for k, str := range strs {
		n, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("invalid index: cannot convert %s: %v", str, err)
		}
		(*i)[k] = n
	}
	return nil
}

func chk(err error) {
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}
