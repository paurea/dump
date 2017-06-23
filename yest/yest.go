package main

import (
	"dump/dnav"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var debug bool

func rdFlags(tIval *dnav.DumpDate) {
	py := flag.Int("y", 0, "# years ago")
	pm := flag.Int("m", 0, "# months ago")
	pd := flag.Int("d", 1, "# days ago")
	ph := flag.Int("h", 0, "# hours ago")
	db := flag.Bool("D", false, "debug flag")
	flag.Parse()
	*tIval = *dnav.NewDumpDate(-*py, -*pm, -*pd, -*ph)

	dnav.Debug = *db
	debug = *db
}

func Dprintf(format string, a ...interface{}) (n int, err error) {

	if !debug {
		return 0, nil
	}
	return fmt.Fprintf(os.Stderr, "yest: "+format, a...)
}

func usage() {
	log.Fatal("yest [-y=n] [-m=n] [-d=n] [-h=n] [-D] file_path")
}

func main() {
	var (
		roots dnav.Roots
		tIval dnav.DumpDate
		path  string
	)

	rdFlags(&tIval)
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
	Dprintf("%s\n", &tIval)
	dnav.RdRoots(&roots)
	Dprintf("mainRoot: %v, dumpRoot: %v, rootName: %v\n", roots.MainRoot, roots.DumpRoot, roots.RootName)

	t := time.Now()
	dDate := dnav.TimeAddDate(t, tIval)
	isD := dnav.IsDump(path, roots)
	if isD {
		dDateDmp, err := dnav.ParseDumpPath(path, roots)
		if err == nil {
			dDate = dnav.SumDates(dDateDmp, tIval)
		}
	}
	Dprintf("date %v\n", dDate)
	yestpath := dnav.FindDumpPath(dDate, roots)
	if yestpath == roots.DumpRoot {
		log.Fatal("could not find dump")
	}
	Dprintf("partial %s\n", yestpath)
	if isD {
		yestpath = yestpath + path[len(yestpath):]
	} else {
		suff := strings.TrimPrefix(path, roots.MainRoot)
		Dprintf("suff %s\n", suff)

		yestpath = yestpath + "/" + roots.RootName + suff
	}
	yestpath = filepath.Clean(yestpath)
	if _, err := os.Stat(yestpath); err != nil {
		fmt.Fprintf(os.Stderr, "path does not exist: %s", err)
	}
	zDate := dnav.DumpDate{}
	if zDate != dDate && strings.HasPrefix(yestpath, path) {
		log.Fatal("could not find previous file in dump")
	}

	fmt.Println(yestpath)

}
