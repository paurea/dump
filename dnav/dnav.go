package dnav

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	MainRootVar     = "MAINROOT"
	MainDumpVar     = "DUMPROOT"
	DefaultRoot     = "/newage/NEWAGE"
	DefaultRootName = "NEWAGE"
	DefaultDump     = "/dump"
)

var Debug bool

//A DumpDate represents a path in the Dump in an abstract way,
// by a description of the time. It is an opaque type without exported fields
type DumpDate struct {
	years  int
	months int
	days   int
	hours  int
}

//NewDumpDate creates a new DumpDate from the description.
func NewDumpDate(years int, months int, days int, hours int) *DumpDate {
	return &DumpDate{years, months, days, hours}
}

func (d *DumpDate) SameYear(d2 *DumpDate) bool {
	return d.years == d2.years
}

func (d *DumpDate) SameMonth(d2 *DumpDate) bool {
	return d.months == d2.months
}

func (d *DumpDate) SameDay(d2 *DumpDate) bool {
	return d.days == d2.days
}
func (d *DumpDate) SameHour(d2 *DumpDate) bool {
	return d.hours == d2.hours
}

func (d *DumpDate) String() string {
	return fmt.Sprintf("y:%d m:%d d:%d h:%d", d.years, d.months, d.days, d.hours)
}

func Dprintf(format string, a ...interface{}) (n int, err error) {

	if !Debug {
		return 0, nil
	}
	return fmt.Fprintf(os.Stderr, "dnav: "+format, a...)
}

func SumDates(d1 DumpDate, d2 DumpDate) (ds DumpDate) {
	ds.years = d1.years + d2.years
	ds.months = d1.months + d2.months
	ds.days = d1.days + d2.days
	ds.hours = d1.hours + d2.hours
	return ds
}

//just to make a comparison, needs to be monotonic
func (d *DumpDate) hoursApprox() int {
	ms := d.years*12 + d.months
	ds := ms*31 + d.days
	hs := ds*24 + d.hours
	return hs
}

func (d *DumpDate) IsAfter(d2 DumpDate) bool {
	h := d.hoursApprox()
	h2 := d2.hoursApprox()
	return h > h2
}

//Convert a time into a DumpDate
func TInDumpDate(t time.Time) DumpDate {
	return DumpDate{t.Year(), int(t.Month()), t.Day(), t.Hour()*100 + t.Minute()}
}

//Add DumpDate to a time to obtain a DumpDate
func TimeAddDate(t time.Time, deltaT DumpDate) DumpDate {
	t2 := t.AddDate(deltaT.years, deltaT.months, deltaT.days)
	hours := time.Duration(deltaT.hours) * time.Hour
	t2 = t2.Add(hours)
	return TInDumpDate(t2)
}

//for a set of paths separated by colons find the first that exists
func firstExists(paths string) string {
	Dprintf("checking [%s]\n", paths)
	lst := strings.Split(paths, ":")
	for _, path := range lst {
		var err error
		if _, err = os.Stat(path); err == nil {
			return path
		}
		Dprintf("[%s] does not exist: %s\n", path, err)
	}
	return ""
}

type Roots struct {
	MainRoot string
	DumpRoot string
	RootName string
}

//RdRoots finds the first path which exists for each of the environment variables.
//If none exist, it sets the default values.
func RdRoots(roots *Roots) {
	mR := os.Getenv(MainRootVar)
	mD := os.Getenv(MainDumpVar)

	if roots.MainRoot = firstExists(mR); roots.MainRoot == "" {
		roots.MainRoot = DefaultRoot
	}

	if roots.DumpRoot = firstExists(mD); roots.DumpRoot == "" {
		roots.DumpRoot = DefaultDump
	}

	if roots.RootName = path.Base(roots.MainRoot); roots.RootName == "" {
		roots.RootName = DefaultRootName
	}
}

//IsDump finds if a path belongs to the dump
func IsDump(path string, roots Roots) bool {
	return strings.HasPrefix(path, roots.DumpRoot)
}

//ParseDumpPath interprets a dump path as a date (years, months, days, hours). The smallest
//values may be missing and interpeted as zero.
func ParseDumpPath(path string, roots Roots) (d DumpDate, err error) {
	pathInDump := strings.TrimPrefix(path, roots.DumpRoot)

	if len(pathInDump) != len(path)-len(roots.DumpRoot) {
		return d, errors.New("bad root")
	}
	lstNames := strings.Split(pathInDump, "/")
	nNames := len(lstNames)
	if nNames > 0 {
		if d.years, err = strconv.Atoi(lstNames[1]); err != nil {
			if lstNames[1] != roots.RootName {
				return d, errors.New("bad year")
			}
		}
	}
	if nNames > 1 {
		if d.months, err = strconv.Atoi(lstNames[2]); err != nil {
			if lstNames[1] != roots.RootName {
				return d, errors.New("bad month")
			}
		}
		d.days = d.months % 100
		d.months = d.months / 100
	}
	if nNames > 2 {
		if d.hours, err = strconv.Atoi(lstNames[3]); err != nil {
			if lstNames[1] != roots.RootName {
				return d, errors.New("bad hour")
			}
		}

	}
	Dprintf("dump date %s\n", &d)
	return d, err
}

//given a path, find the biggest numeric name smaller than a number
func biggestSmallerEqthan(path string, max int) (curr int, err error) {
	var files []os.FileInfo
	Dprintf("big smaller than %d, %s\n", max, path)
	files, err = ioutil.ReadDir(path)
	if err != nil {
		return -1, err
	}
	curr = -1
	//they are sorted by filename
	for _, file := range files {
		switch file.Name() {
		case "current", "current_chk", "first", "lost+found":
			continue
		default:
			break
		}
		n := 0
		if n, err = strconv.Atoi(file.Name()); err == nil {
			//Dprintf("n:%d, max:%d curr: %d\n", n, max, curr)
			if n <= max && n >= curr {
				curr = n
			}
		} else {
			break
		}
	}
	if err != nil || curr < 0 {
		return -1, errors.New("non numeric file name")
	}
	Dprintf("big smaller than is: %d\n", curr)
	return curr, nil
}

//FindDumpPath looks for a path as close as possible to the
//date, but which may be equal or smaller
func FindDumpPath(d DumpDate, roots Roots) string {
	partial := roots.DumpRoot
	year, err := biggestSmallerEqthan(partial, d.years)
	if err != nil {
		return partial
	}
	partial += "/" + fmt.Sprintf("%4.4d", year)

	month, err := biggestSmallerEqthan(partial, 100*d.months+d.days)
	if err != nil {
		return partial
	}
	partial += "/" + fmt.Sprintf("%4.4d", month)

	hour, err := biggestSmallerEqthan(partial, d.hours)
	if err != nil {
		return partial
	}
	partial += "/" + fmt.Sprintf("%4.4d", hour)
	return partial
}
