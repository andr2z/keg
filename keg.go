package keg

import (
	"bufio"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	Z "github.com/rwxrob/bonzai/z"
	"github.com/rwxrob/fs"
	_fs "github.com/rwxrob/fs"
	"github.com/rwxrob/fs/dir"
	"github.com/rwxrob/fs/file"
	"github.com/rwxrob/keg/kegml"
	"github.com/rwxrob/to"
)

// NodePaths returns a list of node directory paths contained in the
// keg root directory path passed. Paths returns are fully qualified and
// cleaned. Only directories with valid integer node IDs will be
// recognized. Empty slice is returned if kegroot doesn't point to
// directory containing node directories with integer names.
//
// The lowest and highest integer names are returned as well to help
// determine what to name a new directory.
//
// File and directories that do not have an integer name will be
// ignored.
var NodePaths = _fs.IntDirs

var LatestDexEntryExp = regexp.MustCompile(
	`^\* (\d\d\d\d-\d\d-\d\d \d\d:\d\d:\d\dZ) \[(.*)\]\(\.\./(\d+)\)$`,
)

// ParseDex parses any input valid for to.String into a Dex pointer.
// FIXME: replace regular expression with pegn.Scanner instead
func ParseDex(in any) (*Dex, error) {
	dex := Dex{}
	s := bufio.NewScanner(strings.NewReader(to.String(in)))
	for line := 1; s.Scan(); line++ {
		f := LatestDexEntryExp.FindStringSubmatch(s.Text())
		if len(f) != 4 {
			return nil, fmt.Errorf("bad line in latest.md: %v", line)
		}
		if t, err := time.Parse(IsoDateFmt, string(f[1])); err != nil {
			return nil, err
		} else {
			if i, err := strconv.Atoi(f[3]); err != nil {
				return nil, err
			} else {
				dex = append(dex, DexEntry{U: t, T: f[2], N: i})
			}
		}
	}
	return &dex, nil
}

// ReadDex reads an existing dex/latest.md dex and returns it.
func ReadDex(kegdir string) (*Dex, error) {
	f := filepath.Join(kegdir, `dex`, `latest.md`)
	buf, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return ParseDex(buf)
}

// ScanDex takes the target path to a keg root directory returns a
// Dex object.
func ScanDex(kegdir string) (*Dex, error) {
	var dex Dex
	dirs, _, _ := NodePaths(kegdir)
	sort.Slice(dirs, func(i, j int) bool {
		_, iinfo := _fs.LatestChange(dirs[i].Path)
		_, jinfo := _fs.LatestChange(dirs[j].Path)
		return iinfo.ModTime().After(jinfo.ModTime())
	})
	for _, d := range dirs {
		_, i := _fs.LatestChange(d.Path)
		title, _ := kegml.ReadTitle(d.Path)
		id, err := strconv.Atoi(d.Info.Name())
		if err != nil {
			continue
		}
		entry := DexEntry{U: i.ModTime().UTC(), T: title, N: id}
		dex = append(dex, entry)
	}
	return &dex, nil
}

// MakeDex calls ScanDex and writes (or overwrites) the output to the
// reserved dex node file within the kegdir passed. File-level
// locking is attempted using the go-internal/lockedfile (used by Go
// itself). Both a friendly markdown file reverse sorted by time of last
// update (latest.md) and a tab-delimited file sorted numerically by
// node ID (nodes.tsv) are created.
func MakeDex(kegdir string) error {
	dex, err := ScanDex(kegdir)
	if err != nil {
		return err
	}

	// markdown is first since reverse chrono of updates is default
	mdpath := filepath.Join(kegdir, `dex`, `latest.md`)
	if err := file.Overwrite(mdpath, dex.MD()); err != nil {
		return err
	}

	tsvpath := filepath.Join(kegdir, `dex`, `nodes.tsv`)
	if err := file.Overwrite(tsvpath, dex.ByID().TSV()); err != nil {
		return err
	}

	return UpdateUpdated(kegdir)
}

// ImportNode moves the nodedir into the KEG directory for the kegid giving
// it the nodeid name. Import will fail if the given nodeid already
// existing the the target KEG.
func ImportNode(from, to, nodeid string) error {
	to = path.Join(to, nodeid)
	if _fs.Exists(to) {
		return _fs.ErrorExists{to}
	}
	return os.Rename(from, to)
}

// UpdateUpdated sets the updated YAML field in the keg info file.
func UpdateUpdated(kegpath string) error {
	kegfile := filepath.Join(kegpath, `keg`)
	updated := UpdatedString(kegpath)
	return file.ReplaceAllString(
		kegfile, `(^|\n)updated:.*(\n|$)`, `${1}updated: `+updated+`${2}`,
	)
}

// Updated parses the most recent change time in the dex/node.md file
// (the first line) and returns the time stamp it contains as
// a time.Time. If a time stamp could not be determined returns time.
func Updated(kegpath string) (*time.Time, error) {
	kegfile := filepath.Join(kegpath, `dex`, `latest.md`)
	str, err := file.FindString(kegfile, IsoDateExpStr)
	if err != nil {
		return nil, err
	}
	t, err := time.Parse(IsoDateFmt, str)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Last parses and returns a DexEntry of the most recently
// updated node from first line of the dex/latest.md file. If cannot
// determine returns nil.
func Last(kegpath string) *DexEntry {
	kegfile := filepath.Join(kegpath, `dex`, `latest.md`)
	lines, err := file.Head(kegfile, 1)
	if err != nil || len(lines) == 0 {
		return nil
	}
	dex, err := ParseDex(lines[0])
	if err != nil {
		return nil
	}
	return &(*dex)[0]
}

// UpdatedString returns Updated time as a string or an empty string if
// there is a error.
func UpdatedString(kegpath string) string {
	u, err := Updated(kegpath)
	if err != nil {
		log.Println(err)
		return ""
	}
	return (*u).Format(IsoDateFmt)
}

// Publish publishes the keg at kegpath location to its distribution
// targets listed in the keg file under "publish." Currently, this only
// involves looking for a .git directory and if found doing a git push.
// Git commit messages are always based on the latest node title without
// any verb.
func Publish(kegpath string) error {
	gitd, err := fs.HereOrAbove(`.git`)
	if err != nil {
		return err
	}
	origd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(origd)
	os.Chdir(filepath.Dir(gitd))
	if err := Z.Exec(`git`, `-C`, kegpath, `pull`); err != nil {
		return err
	}
	if err := Z.Exec(`git`, `-C`, kegpath, `add`, `-A`, `.`); err != nil {
		return err
	}
	msg := "Publish changes"
	if n := Last(kegpath); n != nil {
		msg = n.T
	}
	if err := Z.Exec(`git`, `-C`, kegpath, `commit`, `-m`, msg); err != nil {
		return err
	}
	return Z.Exec(`git`, `-C`, kegpath, `push`)
}

// MakeNode examines the keg at kegpath for highest integer identifier
// and provides a new one returning a *DexEntry for it.
func MakeNode(kegpath string) (*DexEntry, error) {
	_, _, high := NodePaths(kegpath)
	if high < 0 {
		high = 0
	}
	high++
	path := filepath.Join(kegpath, strconv.Itoa(high))
	if err := dir.Create(path); err != nil {
		return nil, err
	}
	readme := filepath.Join(kegpath, `dex`, `README.md`)
	if err := file.Touch(readme); err != nil {
		return nil, err
	}
	return &DexEntry{N: high}, nil
}

// Edit calls file.Edit on the given node README.md file within the
// given kegpath.
func Edit(kegpath string, id int) error {
	node := strconv.Itoa(id)
	if node == "" {
		return fmt.Errorf(`node (%q) is not a valid node id`, id)
	}
	readme := filepath.Join(kegpath, node, `README.md`)
	return file.Edit(readme)
}

// DexUpdate first checks the keg at kegpath for an existing
// dex/latest.md file and if found loads it, if not, MakeDex is called
// to create it. Then DexUpdate examines the Dex for the DexEntry passed
// and if found updates it with the new information, otherwise, it will
// add the new entry without any further validation. The updated Dex is
// then written to the dex/latest.md file.
func DexUpdate(kegpath string, entry *DexEntry) error {
	if !HaveDex(kegpath) {
		if err := MakeDex(kegpath); err != nil {
			return err
		}
	}
	entry.Update(kegpath)
	dex, err := ReadDex(kegpath)
	if err != nil {
		return err
	}
	found := dex.Lookup(entry.N)
	if found == nil {
		dex.Add(entry)
	} else {
		found.U = entry.U
		found.T = entry.T
	}
	return WriteDex(kegpath, dex)
}

// Lookup does a linear search through the Dex for one with the passed
// id and if found returns, otherwise returns nil.
func (d Dex) Lookup(id int) *DexEntry {
	for _, i := range d {
		if i.N == id {
			return &d[id]
		}
	}
	return nil
}

// HaveDex returns true if keg at kegpath has a dex/latest.md file.
func HaveDex(kegpath string) bool {
	return file.Exists(filepath.Join(kegpath, `dex`, `latest.md`))
}

// WriteDex writes the dex/latest.md and dex/nodes.tsv files to the keg
// at kegpath and calls UpdateUpdated to keep keg info file in sync.
func WriteDex(kegpath string, dex *Dex) error {
	latest := filepath.Join(kegpath, `dex`, `latest.md`)
	nodes := filepath.Join(kegpath, `dex`, `nodes.tsv`)
	if err := file.Overwrite(latest, dex.ByLatest().MD()); err != nil {
		return err
	}
	if err := file.Overwrite(nodes, dex.ByID().TSV()); err != nil {
		return err
	}
	return UpdateUpdated(kegpath)
}

//go:embed testdata/samplekeg/1/README.md
var SampleNodeReadme string

// WriteSample writes the embedded SampleNodeReadme to the entry
// indicated in the keg specified by kegpath.
func WriteSample(kegpath string, entry *DexEntry) error {
	return file.Overwrite(
		filepath.Join(kegpath, entry.ID(), `README.md`),
		SampleNodeReadme,
	)
}
