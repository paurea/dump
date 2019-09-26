package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/paurea/dump/dnav"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	debug        bool
	mChangesFlag bool
	txtFlag      bool

	verbose bool

	yearly  bool
	monthly bool
	daily   bool
	hourly  bool

	earliestPath string
)

func rdFlags() {
	db := flag.Bool("D", false, "debug flag")

	v := flag.Bool("v", false, "verbose flag")
	c := flag.Bool("c", false, "changes, no diffs flag")
	t := flag.Bool("t", false, "txt flag")

	y := flag.Bool("y", false, "filter yearly")
	m := flag.Bool("m", false, "filter monthly")
	d := flag.Bool("d", false, "filter daily")
	h := flag.Bool("h", true, "filter hourly")

	s := flag.String("s", "", "earliest path")
	flag.Parse()

	debug = *db
	dnav.Debug = *db
	mChangesFlag = *c
	txtFlag = *t

	verbose = *v

	yearly = *y
	monthly = *m
	daily = *d
	hourly = *h

	earliestPath = *s

}

func Dprintf(format string, a ...interface{}) (n int, err error) {

	if !debug {
		return 0, nil
	}
	return fmt.Fprintf(os.Stderr, "hist: "+format, a...)
}

func usage() {
	log.Fatal("hist [-Dvc] [-ymdh] [-s=earliestPath] file_path")
}

type pathDump struct {
	path string
	os.FileInfo
}

func pathsBeforeRec(dDate dnav.DumpDate, roots *dnav.Roots, root string, from dnav.DumpDate) (paths []string, err error) {

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}
	found := 0
	for _, f := range files {
		fName := f.Name()
		if _, err := strconv.Atoi(fName); err == nil {
			found++
			pathRec := root+"/"+fName
			d, err := dnav.ParseDumpPath(pathRec, *roots)
			if err != nil || (&d).IsBeforeSparse(from) {
				continue
			}
			files, err := pathsBeforeRec(dDate, roots, pathRec, from)
			if err == nil {
				paths = append(paths, files...)
			}
		}
	}
	if found == 0 {
		return []string{root}, nil
	}

	return paths, nil
}

func pathsBeforeFrom(dDate dnav.DumpDate, from dnav.DumpDate, roots dnav.Roots) (paths []string, err error) {
	Dprintf("pathsBeforeFrom\n")
	paths, err = pathsBeforeRec(dDate, &roots, roots.DumpRoot, from)
	ps := paths
	paths = nil
	lastD := dDate
	Dprintf("filtering paths\n")
	for _, p := range ps {
		d, err2 := dnav.ParseDumpPath(p, roots)
		if err2 != nil {
			return nil, err2
		}
		if (&d).IsBefore(from) {
			continue
		}

		if (&d).IsAfter(dDate) {
			continue
		}
		if yearly && d.SameYear(&lastD) {
			continue
		}
		if monthly && d.SameMonth(&lastD) {
			continue
		}
		if daily && d.SameDay(&lastD) {
			continue
		}
		if hourly && d.SameHour(&lastD) {
			continue
		}
		paths = append(paths, p)
		lastD = d
	}
	return paths, nil
}

func fmtDiff(diffs []diffmatchpatch.Diff, curr *File, new *File) string {
	var buff bytes.Buffer
	nlRight := 0
	nlLeft := 0
	Dprintf("\nDIFFs--\n")
	for _, diff := range diffs {
		text := diff.Text
		nlDiff := strings.Count(text, "\n")
		if len(text) == 0 {
			continue
		}
		diffStr := ""
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			Dprintf(">--insert %d, (%d %d)\n", nlDiff, nlLeft, nlRight)
			buff.WriteString(fmt.Sprintf("\n%s:%d,%d %s:%d,%d\n", curr.path, nlLeft+1, nlLeft+1, new.path, nlRight+1, nlRight+nlDiff+1))
			for i := nlRight; i < nlRight+nlDiff-1; i++ {
				diffStr += ">" + new.lines[i] + "\n"
			}
			if nlDiff == 0 {
				//diffStr += "^" + text + "^" + "\n"
				diffStr += ">" + new.lines[nlRight] + "\n" //TODO: SHOULD MERGE continuous runs
			}
			nlRight += nlDiff
		case diffmatchpatch.DiffDelete:
			Dprintf("<--delete %d, (%d %d)\n", nlDiff, nlLeft, nlRight)
			buff.WriteString(fmt.Sprintf("\n%s:%d,%d %s:%d,%d\n", curr.path, nlLeft+1, nlLeft+nlDiff+1, new.path, nlRight+1, nlRight+1))
			for i := nlLeft; i < nlLeft+nlDiff; i++ {
				diffStr += "<" + curr.lines[i] + "\n"
			}
			if nlDiff == 0 {
				//diffStr += "^" + text + "^" + "\n"
				diffStr += "<" + curr.lines[nlLeft] + "\n" //TODO: SHOULD MERGE continuous runs
			}
			nlLeft += nlDiff
		case diffmatchpatch.DiffEqual:
			Dprintf("=--match %d, (%d %d)\n", nlDiff, nlLeft, nlRight)
			nlLeft += nlDiff
			nlRight += nlDiff
		}
		if diff.Type != diffmatchpatch.DiffEqual {
			_, _ = buff.WriteString("\n" + diffStr)
		}

	}

	return buff.String()
}

type File struct {
	path  string
	lines []string
	txt   string
	sha   [20]byte
	info  os.FileInfo
}

func (f *File) String() string {
	s := fmt.Sprintf("%s ", f.path)
	if f.info == nil {
		s += " ###bad info###"
		return s
	}
	s += fmt.Sprintf("%d ", f.info.Size())
	s += fmt.Sprintf("%#o ", f.info.Mode())
	s += fmt.Sprintf("%v ", f.info.ModTime())
	if f.info.IsDir() {
		s += fmt.Sprintf(" d")
	} else {
		s += fmt.Sprintf(" f")
	}

	return s
}

func (f *File) isText() bool {
	return !strings.ContainsRune(f.txt, utf8.RuneError)
}

func (f *File) isDir() bool {
	return f.info != nil && f.info.IsDir()
}

func (f *File) readDir() (txt string, err error) {
	files, err := ioutil.ReadDir(f.path)
	if err != nil {
		return "", err
	}
	for _, fi := range files {
		fp := &File{path: fi.Name(), info: fi}
		txt += fmt.Sprintf("\t[]\t%s\n", fp)
	}
	return txt, nil
}

func readFile(path string) (f *File, exists bool, err error) {
	var buf []byte
	f = &File{path: path, txt: ""}
	f.info, err = os.Stat(f.path)
	if os.IsNotExist(err) {
		return f, false, nil
	}
	if err != nil {
		return f, false, err
	}
	exists = true
	if f.info.IsDir() {
		f.txt, err = f.readDir()
		if err != nil {
			return f, exists, err
		}
		f.sha = sha1.Sum([]byte(f.txt)) //BETTER WAYS md5? no sec concern here, which is faster?
	} else {
		buf, err = ioutil.ReadFile(path)
		if err == nil {
			exists = true
			f.txt = string(buf)
			f.sha = sha1.Sum(buf) //BETTER WAYS? no sec concern here, which is faster?
		} else if os.IsNotExist(err) {
			return f, false, nil
		}
	}
	f.lines = strings.Split(f.txt, "\n")
	return f, exists, err
}

func (f *File) hasEqContent(f2 *File) bool {
	return bytes.Compare(f.sha[:], f2.sha[:]) == 0
}

func doDiffs(paths []string, dPath string) {
	var err error

	onlyChanges := mChangesFlag

	suff := dPath[len(paths[0]):]
	var curr *File
	new := &File{}

	exists := false
	newexists := false

	j := 0
	for ; j < len(paths); j++ {
		curr, newexists, err = readFile(paths[j] + suff)
		if !newexists {
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n", curr.path, err)
			continue
		}
		if !mChangesFlag && !txtFlag {
			isBin := !curr.isText()
			onlyChanges = isBin
			if isBin {
				Dprintf("binary file %s\n", curr.path)
			}

		}
		fmt.Printf("#create\t%s\n", curr)
		if curr.isDir() && verbose {
			fmt.Printf("%s\n", curr.txt)
		}
		exists = true
		break
	}
	currMeta := fmt.Sprintf("%s", curr)

	for i := j; i < len(paths); i++ {
		*new = *curr
		dmp := diffmatchpatch.New()
		newexists = false
		new, newexists, err = readFile(paths[i] + suff)

		if !newexists && exists {
			fmt.Printf("#delete\t%s -> %s\n", curr.path, new.path)
			exists = false
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s\n", new.path, err)
			continue
		}
		if !exists && newexists {
			fmt.Printf("#create\t%s\n", new)
		}
		exists = newexists
		if !exists {
			continue
		}
		if !mChangesFlag && !txtFlag {
			isBin := !new.isText()
			onlyChanges = isBin
			if isBin {
				Dprintf("binary file %s\n", new.path)
			}
		}

		newMeta := fmt.Sprintf("%s", new)

		if !curr.hasEqContent(new) {
			if onlyChanges {
				fmt.Printf("#write\t%s\n", newMeta)
			}
			if new.isDir() && verbose {
				fmt.Printf("#write\t%s\n", newMeta)
				fmt.Printf("%s\n", new.txt)
			}
			diffs := dmp.DiffMain(curr.txt, new.txt, true)
			diffs = dmp.DiffCleanupSemantic(diffs)
			if !onlyChanges {
				fmt.Println(fmtDiff(diffs, curr, new))
			}
		} else if currMeta[len(paths[i]):] != newMeta[len(paths[i]):] {
			//using os.SameFile here is not what I want, I want only the metada *I* regularly change
			fmt.Printf("#wstat\t%s\n", newMeta)
		}
		curr = new
		currMeta = newMeta
	}
}

func main() {
	var (
		roots    dnav.Roots
		path     string
		fromDate dnav.DumpDate
	)

	rdFlags()
	args := flag.Args()
	if len(flag.Args()) != 1 {
		usage()
	}
	path = args[0]
	if path == "" {
		path = "."
	}
	if path[0] != '/' {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		path = dir + "/" + path
	}
	path = filepath.Clean(path)
	Dprintf("path %s\n", path)
	dnav.RdRoots(&roots)
	Dprintf("mainRoot: %v, dumpRoot: %v, rootName: %v\n", roots.MainRoot, roots.DumpRoot, roots.RootName)

	if earliestPath != "" {
		var err error
		fromDate, err = dnav.ParseDumpPath(earliestPath, roots)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bad earliest dump path")
			usage()
		}
		Dprintf("earliest path %s, earliest date %s\n", earliestPath, fromDate)
	}

	t := time.Now()
	dDate := dnav.TInDumpDate(t)
	Dprintf("date %v\n", dDate)
	isD := dnav.IsDump(path, roots)
	Dprintf("path comes from dump: %v\n", isD)
	if isD {
		dDateDmp, err := dnav.ParseDumpPath(path, roots)
		if err == nil {
			dDate = dDateDmp
			Dprintf("dump date %v\n", dDateDmp)
		}
	}
	dPath := dnav.FindDumpPath(dDate, roots)
	if dPath == roots.DumpRoot {
		log.Fatal("could not find dump")
	}
	Dprintf("partial %s\n", dPath)
	Dprintf("path %s\n", path)
	if isD {
		dPath = dPath + path[len(dPath):]
	} else {
		suff := strings.TrimPrefix(path, roots.MainRoot)
		Dprintf("suff %s\n", suff)

		dPath = dPath + "/" + roots.RootName + suff
	}
	dPath = filepath.Clean(dPath)
	Dprintf("clean dPath %s\n", dPath)
	Dprintf("pathsBeforeFrom dDate %s, fromDate %s, roots %s\n", &dDate, &fromDate, roots)
	files, err := pathsBeforeFrom(dDate, fromDate, roots)
	if err != nil {
		log.Fatal(err)
	}
	Dprintf(" %s: %s\n", files, dPath)
	doDiffs(files, dPath)
}
